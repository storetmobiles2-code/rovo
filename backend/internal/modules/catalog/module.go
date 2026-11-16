// Package catalog owns restaurant onboarding, profile and hours, menu categories/items/variants/
// add-on groups, availability and public discovery/search (charter §3, doc 05).
//
// Implemented: the public read side — cuisines, discovery for a pin, restaurant detail, menu and
// search. Partner and admin write endpoints (onboarding, menu editing) are not implemented yet.
package catalog

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/modules/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	pgeo "github.com/storetmobiles2-code/rovo/backend/internal/platform/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/money"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the catalog module's public API (charter §5).
type Service interface {
	// Cuisines lists active cuisines; with a city, only those served by a live restaurant there.
	Cuisines(ctx context.Context, cityID uuid.UUID) ([]Cuisine, error)
	// Discover returns the restaurants that deliver to a pin.
	Discover(ctx context.Context, q DiscoverQuery) (DiscoverResult, error)
	// Restaurant returns a live restaurant's public profile; with a pin it includes distance,
	// fee, ETA and serviceability for that pin.
	Restaurant(ctx context.Context, restaurantID uuid.UUID, pin *pgeo.Point) (RestaurantDetail, error)
	// Menu returns a live restaurant's current menu.
	Menu(ctx context.Context, restaurantID uuid.UUID) (Menu, error)
	// Search finds restaurants and dishes by name among the restaurants that deliver to a pin.
	Search(ctx context.Context, text string, pin pgeo.Point) (SearchResult, error)
}

// Availability is why a restaurant can or cannot take an order for a pin right now.
type Availability string

// Availabilities (schema RestaurantCard).
const (
	AvailOpen   Availability = "OPEN"
	AvailClosed Availability = "CLOSED"
	AvailPaused Availability = "PAUSED"
	AvailTooFar Availability = "TOO_FAR"
)

// Cuisine is schema Cuisine.
type Cuisine struct {
	Code     string            `json:"code"`
	Name     string            `json:"name"`
	NameI18n map[string]string `json:"nameI18n"`
}

// Rating is a restaurant's aggregate rating; Avg is nil until it has ratings.
type Rating struct {
	Avg   *float64 `json:"avg"`
	Count int      `json:"count"`
}

// ETA is the displayed delivery window.
type ETA struct {
	MinMinutes int `json:"minMinutes"`
	MaxMinutes int `json:"maxMinutes"`
}

// RestaurantCard is schema RestaurantCard. ETA, DeliveryFee and DistanceM are present only when
// the card was computed for a pin the restaurant delivers to.
type RestaurantCard struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	NameI18n     map[string]string `json:"nameI18n"`
	Image        *struct{}         `json:"image"` // media uploads are not built yet; always null
	Cuisines     []string          `json:"cuisines"`
	DietType     string            `json:"dietType"`
	Rating       Rating            `json:"rating"`
	CostForTwo   *money.Money      `json:"costForTwo"`
	ETA          *ETA              `json:"eta,omitempty"`
	DeliveryFee  *money.Money      `json:"deliveryFee,omitempty"`
	DistanceM    *int              `json:"distanceM,omitempty"`
	IsOpen       bool              `json:"isOpen"`
	OpensAt      *string           `json:"opensAt"`
	Availability Availability      `json:"availability"`
	OfferBadge   *string           `json:"offerBadge"`

	cuisineCodes []string
}

// DaySlots are one weekday's opening windows.
type DaySlots struct {
	DayOfWeek int        `json:"dayOfWeek"`
	Slots     []HourSlot `json:"slots"`
}

// HourSlot is one opening window in local "HH:MM".
type HourSlot struct {
	OpensAt  string `json:"opensAt"`
	ClosesAt string `json:"closesAt"`
}

// RestaurantDetail is schema RestaurantDetail.
type RestaurantDetail struct {
	RestaurantCard
	Description    *string                 `json:"description"`
	FSSAILicenseNo *string                 `json:"fssaiLicenseNo"`
	AddressLine    string                  `json:"addressLine"`
	Landmark       *string                 `json:"landmark"`
	PinCode        string                  `json:"pinCode"`
	Location       pgeo.Point              `json:"location"`
	Serviceability *geo.ServiceabilityView `json:"serviceability,omitempty"`
	Hours          []DaySlots              `json:"hours"`
	MenuVersion    int                     `json:"-"`
}

// DiscoverQuery is a discovery request.
type DiscoverQuery struct {
	Pin       pgeo.Point
	Cuisine   string // cuisine code; "" for any
	VegOnly   bool
	MinRating float64
	MaxETAMin int
	HasOffer  bool
	Sort      string // relevance | rating | eta | costLow | costHigh
	Offset    int
	Limit     int
	Locale    string
}

// ZoneNotice tells the client the drop zone is paused (browse allowed, ordering blocked).
type ZoneNotice struct {
	Paused  bool    `json:"paused"`
	Message *string `json:"message"`
}

// DiscoverResult is one page of restaurant cards.
type DiscoverResult struct {
	Items      []RestaurantCard
	NextOffset int // -1 when there are no more
	Zone       *ZoneNotice
}

// SearchResult is the search response.
type SearchResult struct {
	Restaurants []RestaurantCard `json:"restaurants"`
	Dishes      []DishHit        `json:"dishes"`
}

// DishHit is a matching dish with the restaurant that sells it.
type DishHit struct {
	Item       MenuItem       `json:"item"`
	Restaurant RestaurantCard `json:"restaurant"`
}

// Menu is schema Menu.
type Menu struct {
	MenuVersion int            `json:"menuVersion"`
	Categories  []MenuCategory `json:"categories"`
}

// MenuCategory is schema MenuCategory.
type MenuCategory struct {
	ID       uuid.UUID         `json:"id"`
	Name     string            `json:"name"`
	NameI18n map[string]string `json:"nameI18n"`
	Items    []MenuItem        `json:"items"`
}

// MenuItem is schema MenuItem.
type MenuItem struct {
	ID               uuid.UUID         `json:"id"`
	Name             string            `json:"name"`
	NameI18n         map[string]string `json:"nameI18n"`
	Description      *string           `json:"description"`
	VegType          string            `json:"vegType"`
	Price            money.Money       `json:"price"`
	Packaging        money.Money       `json:"packaging"`
	Image            *struct{}         `json:"image"`
	IsAvailable      bool              `json:"isAvailable"`
	UnavailableUntil *time.Time        `json:"unavailableUntil"`
	IsRecommended    bool              `json:"isRecommended"`
	SpiceLevel       *int              `json:"spiceLevel"`
	Serves           *int              `json:"serves"`
	Variants         []ItemVariant     `json:"variants"`
	AddonGroups      []AddonGroup      `json:"addonGroups"`

	categoryID   uuid.UUID
	restaurantID uuid.UUID
}

// ItemVariant is schema ItemVariant.
type ItemVariant struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	NameI18n    map[string]string `json:"nameI18n"`
	Price       money.Money       `json:"price"`
	IsDefault   bool              `json:"isDefault"`
	IsAvailable bool              `json:"isAvailable"`
}

// AddonGroup is schema AddonGroup.
type AddonGroup struct {
	ID        uuid.UUID         `json:"id"`
	Name      string            `json:"name"`
	NameI18n  map[string]string `json:"nameI18n"`
	MinSelect int               `json:"minSelect"`
	MaxSelect int               `json:"maxSelect"`
	Addons    []Addon           `json:"addons"`
}

// Addon is schema Addon.
type Addon struct {
	ID          uuid.UUID         `json:"id"`
	Name        string            `json:"name"`
	NameI18n    map[string]string `json:"nameI18n"`
	Price       money.Money       `json:"price"`
	VegType     string            `json:"vegType"`
	IsAvailable bool              `json:"isAvailable"`
}

type service struct {
	deps      *app.Deps
	cursorKey []byte
}

// New constructs the catalog module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	if deps.Config != nil {
		s.cursorKey = deps.Config.DeriveKey("cursor")
	}
	return &app.Module{Name: app.ServiceCatalog, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}

func (s *service) geo() (geo.Service, error) {
	g, ok := s.deps.Service(app.ServiceGeo).(geo.Service)
	if !ok {
		return nil, apperr.Unavailable("", "Location service unavailable.")
	}
	return g, nil
}

// coverageFor resolves a pin and turns "we do not deliver there" into the 422 the discovery
// endpoints return. A paused zone is not an error: browsing stays allowed.
func (s *service) coverageFor(ctx context.Context, pin pgeo.Point) (geo.Coverage, error) {
	g, err := s.geo()
	if err != nil {
		return geo.Coverage{}, err
	}
	cov, err := g.Resolve(ctx, pin)
	if err != nil {
		return geo.Coverage{}, err
	}
	switch cov.Status {
	case geo.StatusOutsideService:
		return cov, apperr.Unprocessable(apperr.CodeOutsideServiceArea, "We don't deliver here yet.")
	case geo.StatusCityNotLive:
		return cov, apperr.Unprocessable(apperr.CodeCityNotLive, "We are not live in this city yet.")
	}
	if cov.Fees == nil {
		return cov, apperr.Unavailable("", "Delivery pricing is not configured for this area.")
	}
	return cov, nil
}

// card builds the public card for a restaurant row. cov is nil when no pin was given. reachable
// is false when the pin is beyond the delivery radius or outside every fee slab.
func (s *service) card(r *restaurantRow, cov *geo.Coverage, pausedZones map[uuid.UUID]bool, now time.Time) (c RestaurantCard, reachable bool) {
	c = RestaurantCard{
		ID: r.id, Name: r.name, NameI18n: r.nameI18n, Cuisines: r.cuisines, cuisineCodes: r.cuisineCodes,
		DietType: r.dietType, Rating: Rating{Count: r.ratingCount},
	}
	if r.ratingCount > 0 {
		avg := math.Round(float64(r.starsSum)/float64(r.ratingCount)*10) / 10
		c.Rating.Avg = &avg
	}
	if r.costForTwoPaise != nil {
		m := money.INR(money.Paise(*r.costForTwoPaise))
		c.CostForTwo = &m
	}

	local := now.In(r.tz)
	open, opensAt := openAt(r.slots, local)
	switch {
	case !r.acceptingOrders, r.pausedUntil != nil && r.pausedUntil.After(now), pausedZones[r.zoneID]:
		c.Availability = AvailPaused
	case r.closedNow || !open:
		c.Availability = AvailClosed
		// During a planned closure the weekly hours do not say when it reopens.
		if !r.closedNow && opensAt != "" {
			c.OpensAt = &opensAt
		}
	default:
		c.Availability = AvailOpen
	}
	c.IsOpen = c.Availability == AvailOpen

	reachable = true
	if cov != nil && cov.Fees != nil {
		straight := pgeo.StraightM(r.location, cov.Point)
		c.DistanceM = &straight
		road := pgeo.RoadM(straight, cov.Fees.RoadFactorMilli)
		fee, inSlab := cov.Fees.DeliveryFee(road)
		if r.cityID != cov.CityID || straight > min(r.maxRadiusM, cov.Fees.MaxServiceableRadiusM) || !inSlab {
			c.Availability, c.IsOpen, c.OpensAt = AvailTooFar, false, nil
			return c, false
		}
		m := money.INR(money.Paise(fee))
		c.DeliveryFee = &m
		lo, hi := cov.Fees.ETARange(r.prepMin, road, local)
		c.ETA = &ETA{MinMinutes: lo, MaxMinutes: hi}
	}
	return c, reachable
}

// reachableCards returns the cards of every restaurant that delivers to the covered pin.
func (s *service) reachableCards(ctx context.Context, cov geo.Coverage) ([]RestaurantCard, error) {
	g, err := s.geo()
	if err != nil {
		return nil, err
	}
	now := s.deps.Clock.Now()
	rows, err := s.loadRestaurants(ctx, restaurantFilter{cityID: &cov.CityID, near: &cov.Point, maxRadiusM: cov.Fees.MaxServiceableRadiusM}, now)
	if err != nil {
		return nil, err
	}
	paused, err := g.PausedZones(ctx, cov.CityID)
	if err != nil {
		return nil, err
	}
	cards := make([]RestaurantCard, 0, len(rows))
	for i := range rows {
		if c, ok := s.card(&rows[i], &cov, paused, now); ok {
			cards = append(cards, c)
		}
	}
	return cards, nil
}

func (s *service) Discover(ctx context.Context, q DiscoverQuery) (DiscoverResult, error) {
	cov, err := s.coverageFor(ctx, q.Pin)
	if err != nil {
		return DiscoverResult{}, err
	}
	cards, err := s.reachableCards(ctx, cov)
	if err != nil {
		return DiscoverResult{}, err
	}

	kept := cards[:0]
	for _, c := range cards {
		switch {
		case q.HasOffer: // promotions are not built yet, so no restaurant has an offer
		case q.Cuisine != "" && !containsFold(c.cuisineCodes, q.Cuisine):
		case q.VegOnly && c.DietType != "PURE_VEG":
		case q.MinRating > 0 && (c.Rating.Avg == nil || *c.Rating.Avg < q.MinRating):
		case q.MaxETAMin > 0 && (c.ETA == nil || c.ETA.MaxMinutes > q.MaxETAMin):
		default:
			kept = append(kept, c)
		}
	}
	sortCards(kept, q.Sort)

	res := DiscoverResult{Items: []RestaurantCard{}, NextOffset: -1}
	if cov.Status == geo.StatusZonePaused {
		res.Zone = &ZoneNotice{Paused: true, Message: cov.View(q.Locale).Message}
	}
	if q.Offset < len(kept) {
		end := min(q.Offset+q.Limit, len(kept))
		res.Items = kept[q.Offset:end]
		if end < len(kept) {
			res.NextOffset = end
		}
	}
	return res, nil
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

// sortCards orders cards for display. Every order ends on the id so paging is stable.
func sortCards(cards []RestaurantCard, by string) {
	// nil values sort last in both directions.
	cmpNil := func(a, b *float64, desc bool) int {
		switch {
		case a == nil && b == nil:
			return 0
		case a == nil:
			return 1
		case b == nil:
			return -1
		case *a == *b:
			return 0
		case (*a < *b) != desc:
			return -1
		}
		return 1
	}
	cost := func(c RestaurantCard) *float64 {
		if c.CostForTwo == nil {
			return nil
		}
		v := float64(c.CostForTwo.AmountPaise)
		return &v
	}
	dist := func(c RestaurantCard) int {
		if c.DistanceM == nil {
			return 0
		}
		return *c.DistanceM
	}
	sort.SliceStable(cards, func(i, j int) bool {
		a, b := cards[i], cards[j]
		var c int
		switch by {
		case "rating":
			c = cmpNil(a.Rating.Avg, b.Rating.Avg, true)
		case "eta":
			if a.ETA != nil && b.ETA != nil {
				c = a.ETA.MinMinutes - b.ETA.MinMinutes
			}
		case "costLow":
			c = cmpNil(cost(a), cost(b), false)
		case "costHigh":
			c = cmpNil(cost(a), cost(b), true)
		default: // relevance: orderable restaurants first, then nearest
			if a.IsOpen != b.IsOpen {
				return a.IsOpen
			}
		}
		if c != 0 {
			return c < 0
		}
		if dist(a) != dist(b) {
			return dist(a) < dist(b)
		}
		return a.ID.String() < b.ID.String()
	})
}

func (s *service) Restaurant(ctx context.Context, id uuid.UUID, pin *pgeo.Point) (RestaurantDetail, error) {
	now := s.deps.Clock.Now()
	rows, err := s.loadRestaurants(ctx, restaurantFilter{ids: []uuid.UUID{id}}, now)
	if err != nil {
		return RestaurantDetail{}, err
	}
	if len(rows) == 0 {
		return RestaurantDetail{}, apperr.NotFound("Restaurant not found.")
	}
	r := &rows[0]
	g, err := s.geo()
	if err != nil {
		return RestaurantDetail{}, err
	}
	paused, err := g.PausedZones(ctx, r.cityID)
	if err != nil {
		return RestaurantDetail{}, err
	}

	d := RestaurantDetail{
		Description: r.description, FSSAILicenseNo: r.fssai, AddressLine: r.addressLine, Landmark: r.landmark,
		PinCode: r.pinCode, Location: r.location, Hours: hoursView(r.slots), MenuVersion: r.menuVersion,
	}
	if pin == nil {
		d.RestaurantCard, _ = s.card(r, nil, paused, now)
		return d, nil
	}
	cov, err := g.Resolve(ctx, *pin)
	if err != nil {
		return RestaurantDetail{}, err
	}
	view := cov.View("en")
	if cov.InZone() && cov.Fees != nil {
		var reachable bool
		d.RestaurantCard, reachable = s.card(r, &cov, paused, now)
		if !reachable {
			view.Status = geo.StatusTooFar
		}
	} else {
		// The pin is outside the service area: show the profile without fee or ETA.
		d.RestaurantCard, _ = s.card(r, nil, paused, now)
	}
	d.Serviceability = &view
	return d, nil
}

func hoursView(slots []slot) []DaySlots {
	out := []DaySlots{}
	for day := 1; day <= 7; day++ {
		ds := DaySlots{DayOfWeek: day, Slots: []HourSlot{}}
		for _, s := range slots {
			if s.day == day {
				ds.Slots = append(ds.Slots, HourSlot{OpensAt: hhmm(s.opens), ClosesAt: hhmm(s.closes)})
			}
		}
		if len(ds.Slots) > 0 {
			out = append(out, ds)
		}
	}
	return out
}

// maxSearchHits caps each list in a search response.
const maxSearchHits = 30

func (s *service) Search(ctx context.Context, text string, pin pgeo.Point) (SearchResult, error) {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return SearchResult{}, apperr.Validation(apperr.FieldError{Field: "q", Code: "REQUIRED", Detail: "is required"})
	}
	if len(text) > 60 {
		return SearchResult{}, apperr.Validation(apperr.FieldError{Field: "q", Code: "TOO_LONG", Detail: "at most 60 characters"})
	}
	cov, err := s.coverageFor(ctx, pin)
	if err != nil {
		return SearchResult{}, err
	}
	cards, err := s.reachableCards(ctx, cov)
	if err != nil {
		return SearchResult{}, err
	}
	sortCards(cards, "relevance")

	res := SearchResult{Restaurants: []RestaurantCard{}, Dishes: []DishHit{}}
	byID := make(map[uuid.UUID]RestaurantCard, len(cards))
	ids := make([]uuid.UUID, 0, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
		ids = append(ids, c.ID)
		match := strings.Contains(strings.ToLower(c.Name), text) || containsSubstr(c.Cuisines, text)
		for _, v := range c.NameI18n {
			match = match || strings.Contains(strings.ToLower(v), text)
		}
		if match && len(res.Restaurants) < maxSearchHits {
			res.Restaurants = append(res.Restaurants, c)
		}
	}
	items, err := s.searchItems(ctx, ids, text, maxSearchHits, s.deps.Clock.Now())
	if err != nil {
		return SearchResult{}, err
	}
	for _, it := range items {
		res.Dishes = append(res.Dishes, DishHit{Item: it, Restaurant: byID[it.restaurantID]})
	}
	return res, nil
}

func containsSubstr(list []string, lowered string) bool {
	for _, x := range list {
		if strings.Contains(strings.ToLower(x), lowered) {
			return true
		}
	}
	return false
}
