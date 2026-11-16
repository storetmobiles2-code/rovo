// Package pricing owns the stateless cart quote: bill lines (item total, packaging, delivery
// fee, platform fee, small-cart fee, discount, GST 5% on food, ROUND_OFF), signed quoteId and
// COD caps (charter §3, doc 16). Per R59 the platform fee line is kept with a zero amount and is
// not shown. Feature engineer B implements the service; this file fixes the contract.
package pricing

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the pricing module's public API (charter §5).
type Service interface {
	// Quote prices a cart and returns a signed, stored quote (10-minute TTL).
	Quote(ctx context.Context, req QuoteRequest) (Quote, error)
}

// BillComponent is a bill-line type (doc 16 §6). PLATFORM_FEE is retained at zero per R59.
type BillComponent string

// Bill components.
const (
	CompItemTotal    BillComponent = "ITEM_TOTAL"
	CompPackaging    BillComponent = "PACKAGING"
	CompDeliveryFee  BillComponent = "DELIVERY_FEE"
	CompPlatformFee  BillComponent = "PLATFORM_FEE" // zero at launch (R59); line not shown
	CompSmallCartFee BillComponent = "SMALL_CART_FEE"
	CompDiscount     BillComponent = "DISCOUNT"
	CompGST          BillComponent = "GST"
	CompRoundOff     BillComponent = "ROUND_OFF"
)

// QuoteRequest is the quote input (doc 11 schema QuoteRequest).
type QuoteRequest struct {
	RestaurantID uuid.UUID
	AddressID    uuid.UUID
	Lines        []QuoteLineInput
	CouponCode   string
	PaymentMethod string
}

// QuoteLineInput is one requested item.
type QuoteLineInput struct {
	ItemID    uuid.UUID
	VariantID uuid.UUID
	AddonIDs  []uuid.UUID
	Quantity  int
}

// BillLine is one priced bill component.
type BillLine struct {
	Component  BillComponent
	LabelI18n  map[string]string
	AmountPaise int64
	Hidden     bool // true for the zero PLATFORM_FEE line (R59)
}

// Quote is the signed, stored price breakdown.
type Quote struct {
	ID         uuid.UUID
	Signature  string
	Lines      []BillLine
	TotalPaise int64
	ExpiresAt  string
}

type service struct{ deps *app.Deps }

// New constructs the pricing module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServicePricing, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Quote(context.Context, QuoteRequest) (Quote, error) {
	return Quote{}, apperr.NotImplemented("pricing.Quote")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
