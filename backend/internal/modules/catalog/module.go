// Package catalog owns restaurant onboarding, profile and hours, menu categories/items/variants/
// add-on groups, availability and public discovery/search (pg_trgm) (charter §3, doc 05).
// Feature engineer A implements the service; this file fixes the contract.
package catalog

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the catalog module's public API (charter §5).
type Service interface {
	// Restaurant returns a restaurant's public profile.
	Restaurant(ctx context.Context, restaurantID uuid.UUID) (Restaurant, error)
	// Menu returns a restaurant's current menu snapshot.
	Menu(ctx context.Context, restaurantID uuid.UUID) (Menu, error)
}

// Restaurant is a public restaurant profile.
type Restaurant struct {
	ID       uuid.UUID
	Name     string
	ZoneID   uuid.UUID
	IsOpen   bool
	MenuVersion int
}

// Menu is a restaurant's menu snapshot.
type Menu struct {
	RestaurantID uuid.UUID
	Version      int
	Categories   []MenuCategory
}

// MenuCategory groups menu items.
type MenuCategory struct {
	ID    uuid.UUID
	Name  string
	Items []MenuItem
}

// MenuItem is a sellable item (veg/non-veg/egg per doc 05).
type MenuItem struct {
	ID        uuid.UUID
	Name      string
	VegType   string
	PricePaise int64
	Available bool
}

type service struct{ deps *app.Deps }

// New constructs the catalog module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceCatalog, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Restaurant(context.Context, uuid.UUID) (Restaurant, error) {
	return Restaurant{}, apperr.NotImplemented("catalog.Restaurant")
}

func (s *service) Menu(context.Context, uuid.UUID) (Menu, error) {
	return Menu{}, apperr.NotImplemented("catalog.Menu")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
