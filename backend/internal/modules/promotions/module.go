// Package promotions owns coupons: admin CRUD, validation (one per order; flat/percent, min
// order, max discount, per-user/global limits, platform/restaurant funded) and redemption
// accounting (charter §3, doc 16). Feature engineer B implements the service.
package promotions

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the promotions module's public API (charter §5).
type Service interface {
	// Validate checks a coupon against an order context and returns the discount it would apply.
	Validate(ctx context.Context, code string, in CouponContext) (CouponResult, error)
}

// CouponContext is the order context a coupon is validated against.
type CouponContext struct {
	UserID         uuid.UUID
	RestaurantID   uuid.UUID
	ZoneID         uuid.UUID
	ItemTotalPaise int64
}

// CouponResult is the outcome of validation.
type CouponResult struct {
	Code          string
	DiscountPaise int64
	FundedBy      string // PLATFORM | RESTAURANT
}

type service struct{ deps *app.Deps }

// New constructs the promotions module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServicePromotions, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Validate(context.Context, string, CouponContext) (CouponResult, error) {
	return CouponResult{}, apperr.NotImplemented("promotions.Validate")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
