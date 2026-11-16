// Package payments owns payment initiation, the hosted-checkout return, webhook verification
// (signature over the raw body → persist → enqueue process-event in one tx) and refunds
// (charter §3, doc 14). Feature engineer B implements the service; this file fixes the public
// contract and the Provider port. The fake aggregator (fakepa) is a self-contained skeleton: a
// hosted-checkout stand-in and an HMAC-signed webhook simulator, the only provider enabled in
// dev/test (config rejects fake in prod).
package payments

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/money"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the payments module's public API (charter §5).
type Service interface {
	// Initiate creates a payment session for an order and returns the hosted-checkout handoff.
	Initiate(ctx context.Context, orderID uuid.UUID, amount money.Money) (Session, error)
	// HandleWebhook verifies and persists a provider event, enqueuing async processing.
	HandleWebhook(ctx context.Context, provider string, rawBody []byte, headers map[string]string) error
	// Refund issues a full/partial refund against a captured payment.
	Refund(ctx context.Context, orderID uuid.UUID, amount money.Money, reasonCode string) (Refund, error)
}

// Session is the hosted-checkout handoff returned to the client (doc 14 §5).
type Session struct {
	PaymentID   uuid.UUID
	Provider    string
	CheckoutURL string
	ExpiresAt   time.Time
}

// Refund records a refund request/result.
type Refund struct {
	ID       uuid.UUID
	OrderID  uuid.UUID
	Amount   money.Money
	Status   string
	ReasonCode string
}

// Provider is the payment-aggregator port (doc 14 §4). Concrete adapters (fake now; Razorpay /
// Cashfree later) implement it; the module never imports an SDK directly.
type Provider interface {
	// Name is the provider key used in the webhook path and config.
	Name() string
	// CreateCheckout builds a hosted-checkout session for an order amount.
	CreateCheckout(ctx context.Context, req CheckoutRequest) (CheckoutResponse, error)
	// VerifySignature checks the webhook signature over the raw body (R25); it must run before
	// the body is parsed.
	VerifySignature(rawBody []byte, headers map[string]string) bool
	// ParseEvent extracts the normalised event after the signature verified.
	ParseEvent(rawBody []byte) (ProviderEvent, error)
	// FetchPayment re-fetches authoritative payment state (fetch-to-confirm, doc 14 §8).
	FetchPayment(ctx context.Context, providerPaymentID string) (ProviderEvent, error)
}

// CheckoutRequest is the input to CreateCheckout.
type CheckoutRequest struct {
	OrderID    uuid.UUID
	OrderCode  string
	AmountPaise int64
	ReturnURL  string
}

// CheckoutResponse is the provider's hosted-checkout handoff.
type CheckoutResponse struct {
	ProviderPaymentID string
	CheckoutURL       string
	ExpiresAt         time.Time
}

// ProviderEvent is the normalised webhook/fetch result (doc 14 §8).
type ProviderEvent struct {
	ProviderPaymentID string
	EventType         string // AUTHORIZED | CAPTURED | FAILED | REFUNDED
	AmountPaise       int64
	ProviderEventID   string
	OccurredAt        time.Time
}

// ProcessEventArgs is the River job enqueued from the webhook handler (doc 14 §8).
type ProcessEventArgs struct {
	PaymentEventID uuid.UUID `json:"paymentEventId"`
}

// Kind implements queue.JobArgs.
func (ProcessEventArgs) Kind() string { return "payments.process_event" }

// PaymentTimeoutArgs cancels an unpaid order after payments.pending_timeout_s (R5).
type PaymentTimeoutArgs struct {
	OrderID uuid.UUID `json:"orderId"`
}

// Kind implements queue.JobArgs.
func (PaymentTimeoutArgs) Kind() string { return "payments.payment_timeout" }

type service struct {
	deps     *app.Deps
	provider Provider
}

// New constructs the payments module bound to the configured provider.
func New(deps *app.Deps, provider Provider) *app.Module {
	s := &service{deps: deps, provider: provider}
	return &app.Module{Name: app.ServicePayments, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Initiate(context.Context, uuid.UUID, money.Money) (Session, error) {
	return Session{}, apperr.NotImplemented("payments.Initiate")
}

func (s *service) HandleWebhook(context.Context, string, []byte, map[string]string) error {
	return apperr.NotImplemented("payments.HandleWebhook")
}

func (s *service) Refund(context.Context, uuid.UUID, money.Money, string) (Refund, error) {
	return Refund{}, apperr.NotImplemented("payments.Refund")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}

// FakePA is the in-house fake aggregator used in dev/test (charter §3). It signs webhooks with
// an HMAC-SHA256 over the raw body using the shared secret, matching the verify path.
type FakePA struct {
	WebhookSecret string
	CheckoutBase  string
}

// NewFakePA builds the fake provider.
func NewFakePA(webhookSecret, checkoutBase string) *FakePA {
	if checkoutBase == "" {
		checkoutBase = "http://localhost:8080/_fakepa/checkout"
	}
	return &FakePA{WebhookSecret: webhookSecret, CheckoutBase: checkoutBase}
}

// Name returns the provider key.
func (FakePA) Name() string { return "fake" }

// CreateCheckout returns a stand-in hosted-checkout URL.
func (f *FakePA) CreateCheckout(_ context.Context, req CheckoutRequest) (CheckoutResponse, error) {
	id := "fakepa_" + uuid.NewString()
	return CheckoutResponse{
		ProviderPaymentID: id,
		CheckoutURL:       f.CheckoutBase + "?pid=" + id + "&order=" + req.OrderCode,
		ExpiresAt:         time.Now().Add(15 * time.Minute),
	}, nil
}

// Sign returns the hex HMAC-SHA256 of body under the webhook secret (used by the simulator).
func (f *FakePA) Sign(body []byte) string {
	m := hmac.New(sha256.New, []byte(f.WebhookSecret))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}

// VerifySignature constant-time compares the x-webhook-signature header against the HMAC.
func (f *FakePA) VerifySignature(rawBody []byte, headers map[string]string) bool {
	got := headers["x-webhook-signature"]
	if got == "" {
		got = headers["X-Webhook-Signature"]
	}
	want := f.Sign(rawBody)
	return hmac.Equal([]byte(got), []byte(want))
}

// ParseEvent is a stub; feature engineer B parses the fake event JSON.
func (f *FakePA) ParseEvent([]byte) (ProviderEvent, error) {
	return ProviderEvent{}, apperr.NotImplemented("fakepa.ParseEvent")
}

// FetchPayment is a stub.
func (f *FakePA) FetchPayment(context.Context, string) (ProviderEvent, error) {
	return ProviderEvent{}, apperr.NotImplemented("fakepa.FetchPayment")
}

var _ Provider = (*FakePA)(nil)
