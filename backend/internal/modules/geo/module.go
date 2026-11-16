// Package geo owns cities, localities, zone polygons (PostGIS), serviceability (zone ∩ 7 km
// straight-line, road distance ×1.3) and the fee slab lookup (charter §3, docs 16/18).
// Feature engineer A implements the service; this file fixes the contract.
package geo

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the geo module's public API (charter §5).
type Service interface {
	// Serviceability resolves a drop point to a zone and delivery status.
	Serviceability(ctx context.Context, lat, lng float64) (Serviceability, error)
}

// ServiceStatus is the serviceability outcome.
type ServiceStatus string

// Serviceability statuses (doc 11 schema Serviceability).
const (
	StatusServiceable      ServiceStatus = "SERVICEABLE"
	StatusOutsideService   ServiceStatus = "OUTSIDE_SERVICE_AREA"
	StatusCityNotLive      ServiceStatus = "CITY_NOT_LIVE"
	StatusZonePaused       ServiceStatus = "ZONE_PAUSED"
	StatusTooFar           ServiceStatus = "TOO_FAR"
)

// Serviceability is the resolved coverage for a point.
type Serviceability struct {
	Status          ServiceStatus
	CityID          uuid.UUID
	ZoneID          uuid.UUID
	DeliveryFeePaise int64
}

type service struct{ deps *app.Deps }

// New constructs the geo module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceGeo, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Serviceability(context.Context, float64, float64) (Serviceability, error) {
	return Serviceability{}, apperr.NotImplemented("geo.Serviceability")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
