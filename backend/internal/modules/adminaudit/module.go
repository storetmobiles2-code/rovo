// Package adminaudit owns the append-only audit log (DB grants + trigger), the maker-checker
// approval queue for the five R31 action families, reason codes, feature flags and scoped
// app_config (RESTAURANT → ZONE → CITY → GLOBAL resolution), per doc 10 §11 and charter §3.
// Backend engineer A implements the service; this file fixes the public contract: the Service
// interface, the DTOs, the event and job-arg types, and the Register / RegisterJobs wiring
// convention (charter §5).
package adminaudit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the adminaudit module's public API. Other modules reach it through
// app.Deps.Service(app.ServiceAdminAudit).
type Service interface {
	// Record appends an audit entry inside the caller's transaction so the entry commits or
	// rolls back with the change it describes.
	Record(ctx context.Context, tx pgx.Tx, e Entry) error
	// ListAudit returns a filtered, cursor-paginated page of audit entries (admin viewer).
	ListAudit(ctx context.Context, f AuditFilter) (AuditPage, error)
	// RequestApproval opens a maker-checker request (R31) and returns it in PENDING.
	RequestApproval(ctx context.Context, tx pgx.Tx, req ApprovalRequest) (Approval, error)
	// Decide approves or rejects a pending request; the checker must differ from the maker
	// unless the request is break-glass.
	Decide(ctx context.Context, d Decision) (Approval, error)
	// Config resolves an app_config key for a scope chain, falling back to GLOBAL.
	Config(ctx context.Context, key string, scope ConfigScope) (json.RawMessage, error)
	// FlagEnabled reports whether a feature flag is on.
	FlagEnabled(ctx context.Context, key string) (bool, error)
	// ReasonCodes lists the active reason codes of a category for an actor type.
	ReasonCodes(ctx context.Context, category, actorType string) ([]ReasonCode, error)
}

// ActorType identifies who performed an audited action (audit_logs.actor_type).
type ActorType string

// Actor types.
const (
	ActorUser            ActorType = "USER"
	ActorAdmin           ActorType = "ADMIN"
	ActorSystem          ActorType = "SYSTEM"
	ActorProviderWebhook ActorType = "PROVIDER_WEBHOOK"
	ActorCLI             ActorType = "CLI"
)

// Outcome is the result of an audited action.
type Outcome string

// Outcomes.
const (
	OutcomeSuccess Outcome = "SUCCESS"
	OutcomeDenied  Outcome = "DENIED"
	OutcomeError   Outcome = "ERROR"
)

// Retention classes (doc 10 §13).
const (
	RetentionSecurity1Y = "SECURITY_1Y"
	RetentionMoney8Y    = "MONEY_8Y"
	RetentionOps2Y      = "OPS_2Y"
)

// Entry is one audit-log row. Changes carries a redacted before/after diff (no raw PII).
type Entry struct {
	OccurredAt     time.Time
	ActorType      ActorType
	ActorID        uuid.UUID
	ActorRoles     []string
	SessionID      uuid.UUID
	RequestID      string
	CityID         uuid.UUID
	Action         string
	ResourceType   string
	ResourceID     uuid.UUID
	Outcome        Outcome
	Reason         string
	ApprovalID     uuid.UUID
	Changes        json.RawMessage
	RetentionClass string
}

// AuditFilter narrows the audit viewer query.
type AuditFilter struct {
	ActorID      uuid.UUID
	Action       string
	ResourceType string
	ResourceID   uuid.UUID
	From, To     time.Time
	Cursor       string
	Limit        int
}

// AuditPage is a cursor page of audit entries.
type AuditPage struct {
	Items      []Entry
	NextCursor string
}

// ActionType is a maker-checker action family member (approval_requests.action_type).
type ActionType string

// Approval action types (R31).
const (
	ActionRefund              ActionType = "REFUND"
	ActionGoodwillCoupon      ActionType = "GOODWILL_COUPON"
	ActionLedgerAdjustment    ActionType = "LEDGER_ADJUSTMENT"
	ActionRiderCashAdjustment ActionType = "RIDER_CASH_ADJUSTMENT"
	ActionPayoutBatch         ActionType = "PAYOUT_BATCH"
	ActionCommissionChange    ActionType = "COMMISSION_CHANGE"
	ActionFeeConfigChange     ActionType = "FEE_CONFIG_CHANGE"
	ActionPayoutAccountChange ActionType = "PAYOUT_ACCOUNT_CHANGE"
	ActionRoleGrant           ActionType = "ROLE_GRANT"
)

// ApprovalStatus is the lifecycle state of an approval request.
type ApprovalStatus string

// Approval statuses.
const (
	ApprovalPending  ApprovalStatus = "PENDING"
	ApprovalApproved ApprovalStatus = "APPROVED"
	ApprovalRejected ApprovalStatus = "REJECTED"
	ApprovalExpired  ApprovalStatus = "EXPIRED"
	ApprovalExecuted ApprovalStatus = "EXECUTED"
	ApprovalFailed   ApprovalStatus = "FAILED"
)

// ApprovalRequest is the maker's input. MakerReason must be at least 15 characters.
type ApprovalRequest struct {
	CityID       uuid.UUID
	ActionType   ActionType
	TargetType   string
	TargetID     uuid.UUID
	Payload      json.RawMessage
	AmountPaise  int64
	MakerID      uuid.UUID
	MakerReason  string
	IsBreakGlass bool
}

// Approval is the stored approval request.
type Approval struct {
	ID          uuid.UUID
	ActionType  ActionType
	Status      ApprovalStatus
	MakerID     uuid.UUID
	CheckerID   uuid.UUID
	AmountPaise int64
	ExpiresAt   time.Time
	Version     int
}

// Decision is the checker's verdict on a pending approval.
type Decision struct {
	ApprovalID uuid.UUID
	CheckerID  uuid.UUID
	Approve    bool
	Reason     string
	Version    int
}

// ConfigScope is the most specific scope to resolve an app_config key for; zero IDs are skipped.
type ConfigScope struct {
	RestaurantID uuid.UUID
	ZoneID       uuid.UUID
	CityID       uuid.UUID
}

// ReasonCode is one catalogue entry (doc 13 §6.3).
type ReasonCode struct {
	Category     string
	Code         string
	Label        string
	RequiresNote bool
	FaultParty   string
}

// Event-type constants published by adminaudit.
const (
	EventApprovalRequested = "adminaudit.approval_requested.v1"
	EventApprovalDecided   = "adminaudit.approval_decided.v1"
	EventConfigChanged     = "adminaudit.config_changed.v1"
)

// ExpireApprovalsArgs expires PENDING approvals past expires_at (periodic).
type ExpireApprovalsArgs struct{}

// Kind implements queue.JobArgs.
func (ExpireApprovalsArgs) Kind() string { return "adminaudit.expire_approvals" }

// ExecuteApprovalArgs runs the approved action through its owning module.
type ExecuteApprovalArgs struct {
	ApprovalID uuid.UUID `json:"approvalId"`
}

// Kind implements queue.JobArgs.
func (ExecuteApprovalArgs) Kind() string { return "adminaudit.execute_approval" }

type service struct{ deps *app.Deps }

// New constructs the adminaudit module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceAdminAudit, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Record(context.Context, pgx.Tx, Entry) error {
	return apperr.NotImplemented("adminaudit.Record")
}

func (s *service) ListAudit(context.Context, AuditFilter) (AuditPage, error) {
	return AuditPage{}, apperr.NotImplemented("adminaudit.ListAudit")
}

func (s *service) RequestApproval(context.Context, pgx.Tx, ApprovalRequest) (Approval, error) {
	return Approval{}, apperr.NotImplemented("adminaudit.RequestApproval")
}

func (s *service) Decide(context.Context, Decision) (Approval, error) {
	return Approval{}, apperr.NotImplemented("adminaudit.Decide")
}

func (s *service) Config(context.Context, string, ConfigScope) (json.RawMessage, error) {
	return nil, apperr.NotImplemented("adminaudit.Config")
}

func (s *service) FlagEnabled(context.Context, string) (bool, error) {
	return false, apperr.NotImplemented("adminaudit.FlagEnabled")
}

func (s *service) ReasonCodes(context.Context, string, string) ([]ReasonCode, error) {
	return nil, apperr.NotImplemented("adminaudit.ReasonCodes")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
