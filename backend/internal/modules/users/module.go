// Package users owns the member profile, saved addresses, consents and the /me surface
// (charter §3, doc 12). Feature engineer A implements the service; this file fixes the contract.
package users

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the users module's public API (charter §5).
type Service interface {
	// Me returns the authenticated member's profile and role contexts.
	Me(ctx context.Context, userID uuid.UUID) (Me, error)
	// ListAddresses returns a member's saved delivery addresses.
	ListAddresses(ctx context.Context, userID uuid.UUID) ([]Address, error)
}

// Me is the /me response shape (doc 11 §2.2).
type Me struct {
	ID              uuid.UUID
	FullName        string
	PhoneMasked     string
	PreferredLocale string
}

// Address is a saved delivery address.
type Address struct {
	ID             uuid.UUID
	HouseNo        string
	BuildingStreet string
	Landmark       string
	LocalityID     uuid.UUID
	PinCode        string
	Lat, Lng       float64
}

type service struct{ deps *app.Deps }

// New constructs the users module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceUsers, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Me(context.Context, uuid.UUID) (Me, error) {
	return Me{}, apperr.NotImplemented("users.Me")
}

func (s *service) ListAddresses(context.Context, uuid.UUID) ([]Address, error) {
	return nil, apperr.NotImplemented("users.ListAddresses")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
