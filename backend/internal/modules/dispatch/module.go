// Package dispatch owns rider onboarding/approval, online/offline + location pings, the two-tier
// candidate search (R34), offers with the 45 s timeout cascade (River) and REVOKED, admin manual
// assign, delivery milestones, the delivery OTP (prepaid ≥ ₹300, R39), COD collection + cash-in-
// hand limit (R6) and undeliverable handling (charter §3, doc 06). Feature engineer C implements
// the service; this file fixes the contract and the offer-timer job-arg type.
package dispatch

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the dispatch module's public API (charter §5).
type Service interface {
	// StartDispatch begins the candidate search and offer cascade for a ready order.
	StartDispatch(ctx context.Context, orderID uuid.UUID) error
	// AcceptOffer assigns a delivery to the accepting rider.
	AcceptOffer(ctx context.Context, offerID, riderID uuid.UUID) (Delivery, error)
}

// Delivery is the delivery aggregate view.
type Delivery struct {
	ID      uuid.UUID
	OrderID uuid.UUID
	RiderID uuid.UUID
	Status  string
	Version int
}

// OfferTimeoutArgs expires an unaccepted offer and advances the cascade (R16/R34).
type OfferTimeoutArgs struct {
	OfferID uuid.UUID `json:"offerId"`
}

// Kind implements queue.JobArgs.
func (OfferTimeoutArgs) Kind() string { return "dispatch.offer_timeout" }

// DispatchRetryArgs re-runs the candidate search when no rider accepted (R34).
type DispatchRetryArgs struct {
	OrderID uuid.UUID `json:"orderId"`
	Attempt int       `json:"attempt"`
}

// Kind implements queue.JobArgs.
func (DispatchRetryArgs) Kind() string { return "dispatch.retry" }

type service struct{ deps *app.Deps }

// New constructs the dispatch module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceDispatch, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) StartDispatch(context.Context, uuid.UUID) error {
	return apperr.NotImplemented("dispatch.StartDispatch")
}

func (s *service) AcceptOffer(context.Context, uuid.UUID, uuid.UUID) (Delivery, error) {
	return Delivery{}, apperr.NotImplemented("dispatch.AcceptOffer")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
