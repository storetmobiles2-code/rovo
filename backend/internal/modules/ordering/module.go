// Package ordering owns order placement (Idempotency-Key, stale-quote 409 diff, COD caps), the
// full state machine (doc 13, R1–R5, R40), the River timers (payment timeout, 180 s accept
// ladder, 60 s auto-PREPARING), status history, the cancellation matrix and the restaurant inbox
// (charter §3). Feature engineer B implements the service; this file fixes the contract and the
// timer job-arg types.
package ordering

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the ordering module's public API (charter §5).
type Service interface {
	// Place creates an order from a signed quote under an idempotency key.
	Place(ctx context.Context, req PlaceRequest) (Order, error)
	// Get returns an order the caller may view.
	Get(ctx context.Context, orderID uuid.UUID) (Order, error)
}

// PlaceRequest is the order-placement input (doc 11 schema CreateOrderRequest).
type PlaceRequest struct {
	QuoteID        uuid.UUID
	IdempotencyKey string
	AddressID      uuid.UUID
	PaymentMethod  string
	CustomerNote   string
}

// Status is an order status (doc 13).
type Status string

// Order statuses (doc 11 schema OrderStatus).
const (
	StatusPendingPayment Status = "PENDING_PAYMENT"
	StatusPlaced         Status = "PLACED"
	StatusAccepted       Status = "ACCEPTED"
	StatusPreparing      Status = "PREPARING"
	StatusReadyForPickup Status = "READY_FOR_PICKUP"
	StatusPickedUp       Status = "PICKED_UP"
	StatusDelivered      Status = "DELIVERED"
	StatusPaymentFailed  Status = "PAYMENT_FAILED"
	StatusRejected       Status = "REJECTED"
	StatusCancelled      Status = "CANCELLED"
	StatusUndeliverable  Status = "UNDELIVERABLE"
)

// Order is the order aggregate view.
type Order struct {
	ID        uuid.UUID
	Code      string
	Status    Status
	Version   int
	TotalPaise int64
}

// Event-type constants published by ordering (doc 13 §8).
const (
	EventOrderPlaced    = "ordering.order_placed.v1"
	EventOrderAccepted  = "ordering.order_accepted.v1"
	EventOrderReady     = "ordering.order_ready.v1"
	EventOrderCancelled = "ordering.order_cancelled.v1"
)

// AcceptLadderArgs drives the 180 s accept ladder (ring/owner/ops escalation, R1).
type AcceptLadderArgs struct {
	OrderID uuid.UUID `json:"orderId"`
	Step    int       `json:"step"`
}

// Kind implements queue.JobArgs.
func (AcceptLadderArgs) Kind() string { return "ordering.accept_ladder" }

// AutoPreparingArgs moves ACCEPTED → PREPARING after 60 s if the restaurant has not (R40).
type AutoPreparingArgs struct {
	OrderID uuid.UUID `json:"orderId"`
}

// Kind implements queue.JobArgs.
func (AutoPreparingArgs) Kind() string { return "ordering.auto_preparing" }

type service struct{ deps *app.Deps }

// New constructs the ordering module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceOrdering, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Place(context.Context, PlaceRequest) (Order, error) {
	return Order{}, apperr.NotImplemented("ordering.Place")
}

func (s *service) Get(context.Context, uuid.UUID) (Order, error) {
	return Order{}, apperr.NotImplemented("ordering.Get")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
