// Package ledger owns the double-entry journal (balance enforced by a deferred DB trigger),
// postings on delivery / COD / refund / coupon, commission plans, rider pay, the weekly
// settlement run → payout batch → maker-checker approval → mark-paid with UTR, and COD deposits
// (doc 14, doc 10 §9, charter §3). Backend engineer C implements the service; this file fixes
// the public contract and the job-arg types.
package ledger

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the ledger module's public API. Other modules reach it through
// app.Deps.Service(app.ServiceLedger).
type Service interface {
	// Post writes one balanced journal inside the caller's transaction. It is idempotent on
	// Journal.IdempotencyKey: a replay returns the existing journal ID.
	Post(ctx context.Context, tx pgx.Tx, j Journal) (uuid.UUID, error)
	// Balance returns an account's current balance (debit-positive paise).
	Balance(ctx context.Context, ref AccountRef) (int64, error)
	// RunSettlement builds a DRAFT payout batch for a payee type and period.
	RunSettlement(ctx context.Context, req SettlementRequest) (PayoutBatch, error)
	// MarkPaid records the bank UTR for a payout in an approved batch.
	MarkPaid(ctx context.Context, payoutID uuid.UUID, utr string, paidAt time.Time) error
	// RecordCODDeposit records cash handed over by a rider, pending finance verification.
	RecordCODDeposit(ctx context.Context, d CODDeposit) (uuid.UUID, error)
}

// AccountCode is a chart-of-accounts code (doc 14 §10.2).
type AccountCode string

// Chart of accounts.
const (
	AccountPAClearing        AccountCode = "PA_CLEARING"
	AccountBank              AccountCode = "BANK"
	AccountRiderCashInHand   AccountCode = "RIDER_CASH_IN_HAND"
	AccountCustomerAdvances  AccountCode = "CUSTOMER_ADVANCES"
	AccountRefundsPayable    AccountCode = "REFUNDS_PAYABLE"
	AccountRestaurantPayable AccountCode = "RESTAURANT_PAYABLE"
	AccountRiderPayable      AccountCode = "RIDER_PAYABLE"
	AccountGSTOutputFood     AccountCode = "GST_OUTPUT_9_5_RESTAURANT"
	AccountGSTOutputDelivery AccountCode = "GST_OUTPUT_9_5_DELIVERY"
	AccountGSTOutputOwn      AccountCode = "GST_OUTPUT_OWN"
	AccountGSTInputCredit    AccountCode = "GST_INPUT_CREDIT"
	AccountTDSPayable        AccountCode = "TDS_194O_PAYABLE"
	AccountRevenueCommission AccountCode = "REVENUE_COMMISSION"
	AccountRevenueDelivery   AccountCode = "REVENUE_DELIVERY_FEE"
	AccountRevenuePlatform   AccountCode = "REVENUE_PLATFORM_FEE"
	AccountExpenseRiderPay   AccountCode = "EXPENSE_RIDER_PAY"
	AccountExpensePAFees     AccountCode = "EXPENSE_PA_FEES"
	AccountExpensePromotions AccountCode = "EXPENSE_PROMOTIONS"
	AccountExpenseGoodwill   AccountCode = "EXPENSE_GOODWILL"
	AccountRoundOff          AccountCode = "ROUND_OFF"
	AccountSuspense          AccountCode = "SUSPENSE"
)

// OwnerType is the party an account belongs to.
type OwnerType string

// Owner types.
const (
	OwnerPlatform   OwnerType = "PLATFORM"
	OwnerRestaurant OwnerType = "RESTAURANT"
	OwnerRider      OwnerType = "RIDER"
)

// AccountRef addresses an account by its natural key; owner accounts are created lazily.
type AccountRef struct {
	CityID     uuid.UUID
	Code       AccountCode
	OwnerType  OwnerType
	OwnerID    uuid.UUID
	GSTINState string
}

// EntryType is a journal entry type (ledger_journals.entry_type).
type EntryType string

// Entry types.
const (
	EntryPaymentCaptured EntryType = "PAYMENT_CAPTURED"
	EntryOrderSettled    EntryType = "ORDER_SETTLED"
	EntryCODCollected    EntryType = "COD_COLLECTED"
	EntryRiderEarning    EntryType = "RIDER_EARNING"
	EntryRefund          EntryType = "REFUND"
	EntryPayout          EntryType = "PAYOUT"
	EntryCODDeposit      EntryType = "COD_DEPOSIT"
	EntryAdjustment      EntryType = "ADJUSTMENT"
	EntryReversal        EntryType = "REVERSAL"
)

// Posting is one journal line: positive AmountPaise is a debit, negative a credit.
type Posting struct {
	Account     AccountRef
	AmountPaise int64
	OrderID     uuid.UUID
	Memo        string
}

// Journal is a balanced set of postings (Σ AmountPaise = 0, at least two lines).
type Journal struct {
	CityID         uuid.UUID
	EntryType      EntryType
	AdjustmentType string
	SourceType     string
	SourceID       uuid.UUID
	Rule           string
	IdempotencyKey string
	Description    string
	OccurredAt     time.Time
	ApprovalID     uuid.UUID
	Postings       []Posting
}

// SettlementRequest selects a payee type and a [PeriodStart, PeriodEnd) window in city time.
type SettlementRequest struct {
	CityID      uuid.UUID
	PayeeType   OwnerType
	PeriodStart time.Time
	PeriodEnd   time.Time
}

// PayoutBatch is one settlement run's batch.
type PayoutBatch struct {
	ID            uuid.UUID
	PayeeType     OwnerType
	Status        string
	PayoutCount   int
	TotalNetPaise int64
	Version       int
}

// CODDeposit is cash a rider handed to the platform.
type CODDeposit struct {
	RiderID     uuid.UUID
	AmountPaise int64
	Method      string
	Reference   string
	DepositedAt time.Time
}

// Event-type constants published by ledger.
const (
	EventPayoutBatchCreated = "ledger.payout_batch_created.v1"
	EventPayoutPaid         = "ledger.payout_paid.v1"
	EventCODDepositVerified = "ledger.cod_deposit_verified.v1"
)

// SettlementRunArgs triggers the weekly settlement run for one payee type (periodic; the
// schedule comes from app_config payouts.*.schedule).
type SettlementRunArgs struct {
	CityID    uuid.UUID `json:"cityId"`
	PayeeType string    `json:"payeeType"`
}

// Kind implements queue.JobArgs.
func (SettlementRunArgs) Kind() string { return "ledger.settlement_run" }

// PostOrderSettlementArgs posts the ORDER_SETTLED / RIDER_EARNING journals after delivery.
type PostOrderSettlementArgs struct {
	OrderID uuid.UUID `json:"orderId"`
}

// Kind implements queue.JobArgs.
func (PostOrderSettlementArgs) Kind() string { return "ledger.post_order_settlement" }

type service struct{ deps *app.Deps }

// New constructs the ledger module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceLedger, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Post(context.Context, pgx.Tx, Journal) (uuid.UUID, error) {
	return uuid.Nil, apperr.NotImplemented("ledger.Post")
}

func (s *service) Balance(context.Context, AccountRef) (int64, error) {
	return 0, apperr.NotImplemented("ledger.Balance")
}

func (s *service) RunSettlement(context.Context, SettlementRequest) (PayoutBatch, error) {
	return PayoutBatch{}, apperr.NotImplemented("ledger.RunSettlement")
}

func (s *service) MarkPaid(context.Context, uuid.UUID, string, time.Time) error {
	return apperr.NotImplemented("ledger.MarkPaid")
}

func (s *service) RecordCODDeposit(context.Context, CODDeposit) (uuid.UUID, error) {
	return uuid.Nil, apperr.NotImplemented("ledger.RecordCODDeposit")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
