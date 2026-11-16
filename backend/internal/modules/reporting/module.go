// Package reporting owns read-side analytics: the restaurant today/week summary, the admin
// dashboard KPIs and asynchronous CSV exports (orders, settlements) stored through platform/blob
// (charter §3, doc 10 §11 report_exports). It reads other modules' tables through read-only
// queries and never writes to them. Backend engineer C implements the service; this file fixes
// the public contract and the job-arg types.
package reporting

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the reporting module's public API. Other modules reach it through
// app.Deps.Service(app.ServiceReporting).
type Service interface {
	// RestaurantSummary returns a restaurant's order and revenue summary for a period.
	RestaurantSummary(ctx context.Context, restaurantID uuid.UUID, period Period) (RestaurantSummary, error)
	// AdminDashboard returns the city-level operational KPIs.
	AdminDashboard(ctx context.Context, cityID uuid.UUID) (Dashboard, error)
	// RequestExport queues a CSV export and returns it in QUEUED.
	RequestExport(ctx context.Context, req ExportRequest) (Export, error)
	// GetExport returns an export; DownloadURL is set once it has SUCCEEDED and not expired.
	GetExport(ctx context.Context, exportID uuid.UUID) (Export, error)
}

// Period is a named reporting window in the city timezone.
type Period string

// Periods.
const (
	PeriodToday Period = "TODAY"
	PeriodWeek  Period = "WEEK"
)

// RestaurantSummary is the restaurant analytics card.
type RestaurantSummary struct {
	Period            Period
	OrderCount        int
	DeliveredCount    int
	RejectedCount     int
	CancelledCount    int
	GrossSalesPaise   int64
	NetPayablePaise   int64
	AvgPrepTimeMin    float64
	AcceptanceRateBps int
}

// Dashboard is the admin KPI snapshot.
type Dashboard struct {
	AsOf                time.Time
	OrdersToday         int
	LiveOrders          int
	UnassignedOrders    int
	RidersOnline        int
	RestaurantsOpen     int
	GMVTodayPaise       int64
	AvgDeliveryTimeMin  float64
	CancellationRateBps int
	OpenTickets         int
}

// ReportType is an export kind (report_exports.report_type).
type ReportType string

// Report types in the V1 slice.
const (
	ReportOrders              ReportType = "ORDERS"
	ReportSettlementStatement ReportType = "SETTLEMENT_STATEMENT"
)

// ExportStatus is the lifecycle of an export.
type ExportStatus string

// Export statuses.
const (
	ExportQueued    ExportStatus = "QUEUED"
	ExportRunning   ExportStatus = "RUNNING"
	ExportSucceeded ExportStatus = "SUCCEEDED"
	ExportFailed    ExportStatus = "FAILED"
	ExportExpired   ExportStatus = "EXPIRED"
)

// ExportRequest is the input for a CSV export.
type ExportRequest struct {
	CityID      uuid.UUID
	ReportType  ReportType
	Params      json.RawMessage
	RequestedBy uuid.UUID
}

// Export is a stored export job.
type Export struct {
	ID          uuid.UUID
	ReportType  ReportType
	Status      ExportStatus
	RowCount    int
	ContainsPII bool
	DownloadURL string
	ExpiresAt   time.Time
}

// GenerateExportArgs builds the CSV for a queued export.
type GenerateExportArgs struct {
	ExportID uuid.UUID `json:"exportId"`
}

// Kind implements queue.JobArgs.
func (GenerateExportArgs) Kind() string { return "reporting.generate_export" }

type service struct{ deps *app.Deps }

// New constructs the reporting module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceReporting, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) RestaurantSummary(context.Context, uuid.UUID, Period) (RestaurantSummary, error) {
	return RestaurantSummary{}, apperr.NotImplemented("reporting.RestaurantSummary")
}

func (s *service) AdminDashboard(context.Context, uuid.UUID) (Dashboard, error) {
	return Dashboard{}, apperr.NotImplemented("reporting.AdminDashboard")
}

func (s *service) RequestExport(context.Context, ExportRequest) (Export, error) {
	return Export{}, apperr.NotImplemented("reporting.RequestExport")
}

func (s *service) GetExport(context.Context, uuid.UUID) (Export, error) {
	return Export{}, apperr.NotImplemented("reporting.GetExport")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
