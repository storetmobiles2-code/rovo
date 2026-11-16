// Command rovo is the single binary for the rovo backend. It provides the
// subcommands the deployment uses: api | worker | all | migrate | seed | version.
//
// The feature HTTP handlers and worker jobs are registered by the domain modules
// under internal/modules; this entrypoint wires the platform (config, logging,
// database, migrations, health, the HTTP middleware chain) and starts the servers.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"log/slog"

	rovoapp "github.com/storetmobiles2-code/rovo/backend/internal/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/config"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/health"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/migrate"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/obs"
	"github.com/storetmobiles2-code/rovo/backend/internal/seed"
)

// version is overridden at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	if cmd == "version" {
		fmt.Println("rovo", version)
		return
	}

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	logger := obs.NewLogger(cfg.LogLevel)
	if len(cfg.Generated) > 0 {
		logger.Warn("generated ephemeral keys (dev/test only); tokens do not survive a restart", "keys", cfg.Generated)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var runErr error
	switch cmd {
	case "migrate":
		runErr = runMigrate(ctx, cfg, logger, os.Args[2:])
	case "seed":
		runErr = runSeed(ctx, cfg, logger)
	case "api":
		runErr = runServers(ctx, cfg, logger, true, false)
	case "worker":
		runErr = runServers(ctx, cfg, logger, false, true)
	case "all":
		runErr = runServers(ctx, cfg, logger, true, true)
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if runErr != nil {
		logger.Error("rovo exited with error", "command", cmd, "err", runErr)
		os.Exit(1)
	}
}

const usage = `rovo <command>

Commands:
  api                 serve the HTTP API (public + internal listeners)
  worker              run background jobs (River) only
  all                 run api and worker in one process (local/dev)
  migrate [up|down|status]   apply, roll back one step, or report migrations (default: up)
  seed                load the Mahabubnagar demo dataset (refused when ROVO_ENV=prod)
  version             print the build version`

func runMigrate(ctx context.Context, cfg *config.Config, logger *slog.Logger, args []string) error {
	action := "up"
	if len(args) > 0 {
		action = args[0]
	}
	r, err := migrate.New(ctx, cfg.DatabaseURL, logger)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer r.Close()

	switch action {
	case "up":
		return r.Up(ctx)
	case "down":
		return r.Down(ctx)
	case "status":
		st, err := r.Status(ctx)
		if err != nil {
			return err
		}
		for _, s := range st {
			mark := "pending"
			if s.Applied {
				mark = "applied"
			}
			fmt.Printf("%-8s %d %s\n", mark, s.Version, s.File)
		}
		return nil
	default:
		return fmt.Errorf("unknown migrate action %q (use up|down|status)", action)
	}
}

// runSeed loads the Mahabubnagar demo dataset (internal/seed). It is idempotent and refuses to
// run when ROVO_ENV=prod. One-time admin credentials are written to stdout, never to the log.
func runSeed(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	if cfg.IsProd() {
		return seed.ErrProd
	}
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, db.Options{MaxConns: 2, StatementTimeout: 30 * time.Second, ApplicationName: "rovo-seed"})
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()

	fields, err := auth.NewFieldKeyring(cfg.FieldKeyID(), map[string][]byte{cfg.FieldKeyID(): cfg.FieldEncryptionKey})
	if err != nil {
		return fmt.Errorf("field keyring: %w", err)
	}
	res, err := seed.Run(ctx, pool, seed.Options{
		Config:    cfg,
		Passwords: auth.NewPasswordHasher(auth.DefaultArgon2, 1),
		Fields:    fields,
		Now:       clock.Real{}.Now(),
	})
	if err != nil {
		return err
	}
	logger.Info("seed complete", "city_id", res.CityID, "zone_id", res.ZoneID, "rows_created", res.Created)

	a := res.Admin
	fmt.Printf("\nSeeded %s (%s), zone %s.\n", seed.CityName, seed.CityCode, seed.ZoneCode)
	fmt.Printf("Admin: %s (ADMIN_SUPER)\n", a.Email)
	switch {
	case a.Password != "":
		fmt.Printf("  password (generated, shown once; must be changed at first login): %s\n", a.Password)
	case a.PasswordFromEnv:
		fmt.Println("  password: set from ROVO_ADMIN_BOOTSTRAP_PASSWORD")
	default:
		fmt.Println("  password: unchanged (set ROVO_ADMIN_BOOTSTRAP_PASSWORD and re-run to reset it)")
	}
	switch {
	case a.TOTPEnrolledAtRun:
		fmt.Printf("  TOTP secret (dev only): %s\n  TOTP URI: %s\n", a.TOTPSecret, a.TOTPURI)
		if slices.Contains(cfg.Generated, "ROVO_FIELD_ENCRYPTION_KEY") {
			fmt.Println("  note: ROVO_FIELD_ENCRYPTION_KEY is unset, so this secret was sealed with an ephemeral key")
			fmt.Println("        that the api process will not have. Set the key in .env and re-run the seed.")
		}
	case cfg.DevLike():
		fmt.Println("  TOTP: unchanged")
	default:
		fmt.Println("  TOTP: not pre-enrolled outside dev/test; the admin enrols at first login")
	}
	return nil
}

// runServers starts the requested servers and blocks until ctx is cancelled, then
// drains connections. The public mux carries the platform middleware chain; feature
// routes are added by module registration through the composition root (internal/app).
func runServers(ctx context.Context, cfg *config.Config, logger *slog.Logger, api, worker bool) error {
	pool, err := db.NewPool(ctx, cfg.DatabaseURL, db.Options{
		MaxConns:         10,
		StatementTimeout: 5 * time.Second,
		ApplicationName:  "rovo-" + serverRole(api, worker),
	})
	if err != nil {
		return fmt.Errorf("db pool: %w", err)
	}
	defer pool.Close()

	application, err := rovoapp.New(cfg, logger, pool, rovoapp.Options{Worker: worker})
	if err != nil {
		return err
	}

	checker := health.New(pool, version)

	// Internal listener: health + (future) metrics. Always on.
	internal := httpx.NewMux()
	internal.Handle("GET /healthz", checker.Live)
	internal.Handle("GET /readyz", checker.Ready)
	internalSrv := &http.Server{Addr: cfg.InternalAddr, Handler: internal, ReadHeaderTimeout: 5 * time.Second}

	var publicSrv *http.Server
	if api {
		mux := httpx.NewMux(
			httpx.RequestContext(logger),
			httpx.Recover(logger),
			httpx.SecurityHeaders(),
			httpx.BodyLimit(1<<20),
		)
		// Every domain module registers its routes on the public mux.
		modules := application.RegisterRoutes(mux)
		logger.Info("modules registered", "count", len(modules), "modules", modules)
		publicSrv = &http.Server{Addr: cfg.HTTPAddr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	}

	errCh := make(chan error, 2)
	go func() {
		logger.Info("internal listener up", "addr", cfg.InternalAddr)
		if err := internalSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("internal server: %w", err)
		}
	}()
	if publicSrv != nil {
		checker.SetListenUp(true)
		go func() {
			logger.Info("api listener up", "addr", cfg.HTTPAddr)
			if err := publicSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("api server: %w", err)
			}
		}()
	}
	if err := application.Start(ctx, api); err != nil {
		return err
	}
	if worker {
		logger.Info("worker enabled", "working_jobs", application.Working())
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received; draining")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if publicSrv != nil {
		_ = publicSrv.Shutdown(shutdownCtx)
	}
	_ = internalSrv.Shutdown(shutdownCtx)
	if err := application.Stop(shutdownCtx); err != nil {
		logger.Warn("river stop", "err", err)
	}
	return nil
}

func serverRole(api, worker bool) string {
	switch {
	case api && worker:
		return "all"
	case worker:
		return "worker"
	default:
		return "api"
	}
}
