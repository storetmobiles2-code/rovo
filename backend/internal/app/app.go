// Package app is the composition root (08 §3.1): it builds the shared platform dependencies
// once, constructs every domain module, publishes the module services into Deps.Services so
// modules can call each other through their exported interfaces, and registers each module's
// HTTP routes and River jobs. cmd/rovo calls it; nothing else imports the modules together.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	blobfake "github.com/storetmobiles2-code/rovo/backend/internal/adapters/blob"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/adminaudit"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/catalog"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/dispatch"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/geo"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/identity"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/ledger"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/notifications"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/ordering"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/payments"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/pricing"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/promotions"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/ratings"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/reporting"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/support"
	"github.com/storetmobiles2-code/rovo/backend/internal/modules/users"
	platform "github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/events"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/idgen"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/ratelimit"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/realtime"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/testhooks"
)

// WorkerQueues are the River queues the worker process serves and their concurrency.
var WorkerQueues = map[string]int{queue.QueueDefault: 10}

// App is the wired application: shared dependencies plus the modules in registration order.
type App struct {
	Deps    *platform.Deps
	Modules []*platform.Module
	Outbox  *testhooks.Outbox

	jobs  *queue.Registry
	queue *queue.Client
	work  bool
}

// Options select what the process runs.
type Options struct {
	// Worker makes the River client work jobs; otherwise it is insert-only (the api process).
	Worker bool
}

// New builds the dependency container and all fifteen modules. It opens no listeners and starts
// no goroutines; call Start for the background pieces.
func New(cfg *config.Config, logger *slog.Logger, pool *pgxpool.Pool, opts Options) (*App, error) {
	clk := clock.Real{}

	keys, err := auth.NewKeyset(cfg.JWTSigningKey)
	if err != nil {
		return nil, fmt.Errorf("app: jwt keyset: %w", err)
	}
	fields, err := auth.NewFieldKeyring(cfg.FieldKeyID(), map[string][]byte{cfg.FieldKeyID(): cfg.FieldEncryptionKey})
	if err != nil {
		return nil, fmt.Errorf("app: field keyring: %w", err)
	}

	bus := realtime.NewBus(pool, logger)
	deps := &platform.Deps{
		Config:      cfg,
		Logger:      logger,
		Pool:        pool,
		Clock:       clk,
		IDs:         idgen.Default{},
		Realtime:    bus,
		RateLimit:   ratelimit.New(pool, clk),
		Blob:        blobfake.New(cfg.S3.PublicMediaBaseURL),
		Tokens:      auth.NewIssuer(keys, cfg.Issuer(), clk),
		Cookies:     auth.Cookies{Secure: cfg.CookieSecure},
		Fields:      fields,
		Passwords:   auth.NewPasswordHasher(auth.DefaultArgon2, 4),
		Idempotency: httpx.NewIdempotencyStore(clk),
		Services:    map[string]any{},
	}

	a := &App{Deps: deps, Outbox: &testhooks.Outbox{}, jobs: queue.NewRegistry(), work: opts.Worker}

	// Modules are constructed against deps before the queue client exists; they must not enqueue
	// from New. Order follows the dependency direction (identity/geo first, read models last).
	a.Modules = []*platform.Module{
		identity.New(deps),
		geo.New(deps),
		users.New(deps),
		catalog.New(deps),
		pricing.New(deps),
		promotions.New(deps),
		ordering.New(deps),
		payments.New(deps, payments.NewFakePA(cfg.FakePAWebhookSecret, "")),
		dispatch.New(deps),
		ledger.New(deps),
		ratings.New(deps),
		notifications.New(deps, notifications.FakeSMS{Outbox: a.Outbox}, notifications.FakePush{Outbox: a.Outbox}),
		support.New(deps),
		adminaudit.New(deps),
		reporting.New(deps),
	}
	for _, m := range a.Modules {
		if _, dup := deps.Services[m.Name]; dup {
			return nil, fmt.Errorf("app: module %q registered twice", m.Name)
		}
		deps.Services[m.Name] = m.Service
	}

	// Job registration happens before the River client is built: the client snapshots the
	// worker set at construction.
	registerPlatformJobs(a.jobs, deps)
	for _, m := range a.Modules {
		if m.Jobs != nil {
			m.Jobs(a.jobs, deps)
		}
	}
	qcfg := queue.Config{}
	if opts.Worker {
		qcfg.Queues = WorkerQueues
	}
	qc, err := queue.New(pool, a.jobs, qcfg)
	if err != nil {
		return nil, fmt.Errorf("app: queue client: %w", err)
	}
	a.queue = qc
	deps.Queue = qc
	deps.Events = events.NewPublisher(events.NewRegistry(), qc, bus)
	return a, nil
}

// RegisterRoutes calls every module's route registrar on the public mux and returns the names
// of the modules registered, in order.
func (a *App) RegisterRoutes(mux *httpx.Mux) []string {
	names := make([]string, 0, len(a.Modules))
	for _, m := range a.Modules {
		if m.Routes != nil {
			m.Routes(mux, a.Deps)
		}
		names = append(names, m.Name)
	}
	return names
}

// Working reports whether this process works River jobs (false for the api-only process).
func (a *App) Working() bool { return a.work }

// Start launches the background pieces: the realtime LISTEN loop when serving the API and the
// River client when working jobs. Both stop when ctx is cancelled / Stop is called.
func (a *App) Start(ctx context.Context, api bool) error {
	if api {
		go a.Deps.Realtime.Run(ctx)
	}
	if a.work {
		if err := a.queue.Start(ctx); err != nil {
			return fmt.Errorf("app: start river: %w", err)
		}
	}
	return nil
}

// Stop drains the River client.
func (a *App) Stop(ctx context.Context) error {
	if a.work {
		return a.queue.Stop(ctx)
	}
	return nil
}
