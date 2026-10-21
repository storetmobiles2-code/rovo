// Package app is the dependency container passed to every module's Register / RegisterJobs. The
// composition root (internal/app) builds it once and wires module services into Services; module
// HTTP and job code reaches shared infrastructure only through these fields (08 §3.1).
package app

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/blob"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/events"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/idgen"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/ratelimit"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/realtime"
)

// Deps are the shared platform dependencies a module receives. Module services are added to
// Services by the composition root after construction so modules can call each other through the
// exported interfaces (never internals).
type Deps struct {
	Config    *config.Config
	Logger    *slog.Logger
	Pool      *pgxpool.Pool
	Clock     clock.Clock
	IDs       idgen.Generator
	Queue     queue.Enqueuer
	Events    *events.Publisher
	Realtime  *realtime.Bus
	RateLimit *ratelimit.Limiter
	Blob      blob.Store
	Tokens    *auth.Issuer
	Cookies   auth.Cookies
	Fields    *auth.FieldKeyring
	Passwords *auth.PasswordHasher
	Idempotency *httpx.IdempotencyStore

	// Services holds the cross-module service interfaces. It is an opaque map keyed by a
	// well-known string (see Service* constants) to avoid an import cycle between platform and
	// the modules; the composition root populates it and modules type-assert what they need.
	Services map[string]any
}

// Service registry keys. Each owning module documents the concrete interface its key resolves to
// (e.g. ServiceGeo → geo.Service). Keeping them here lets module A depend on module B's interface
// without platform importing either.
const (
	ServiceIdentity      = "identity"
	ServiceGeo           = "geo"
	ServiceUsers         = "users"
	ServiceCatalog       = "catalog"
	ServicePricing       = "pricing"
	ServicePromotions    = "promotions"
	ServiceOrdering      = "ordering"
	ServicePayments      = "payments"
	ServiceDispatch      = "dispatch"
	ServiceLedger        = "ledger"
	ServiceRatings       = "ratings"
	ServiceNotifications = "notifications"
	ServiceSupport       = "support"
	ServiceAdminAudit    = "adminaudit"
	ServiceReporting     = "reporting"
)

// Service returns the registered service for key, or nil. Callers assert the concrete type.
func (d *Deps) Service(key string) any {
	if d.Services == nil {
		return nil
	}
	return d.Services[key]
}

// Register wires a module's HTTP routes. Each module exposes a Register(mux, deps) matching this
// shape (charter §5). The audience-scoped mux is prepared by the composition root.
type Register func(mux *httpx.Mux, deps *Deps)

// RegisterJobs wires a module's River workers and periodic jobs. Each module exposes a
// RegisterJobs(reg, deps) of this shape.
type RegisterJobs func(reg *queue.Registry, deps *Deps)

// Module is what every module's New returns: its service plus its route and job registrars. The
// composition root collects modules, publishes their services into Deps.Services, then calls
// Routes and Jobs.
type Module struct {
	Name     string
	Service  any
	Routes   Register
	Jobs     RegisterJobs
}
