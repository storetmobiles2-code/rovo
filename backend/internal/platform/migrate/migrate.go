// Package migrate applies River's schema (rivermigrate) and rovo's goose migrations. It runs
// only from `rovo migrate` (a one-off job with owner credentials) and from tests; app processes
// never migrate on boot (doc 10 §16).
package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/storetmobiles2-code/rovo/backend/migrations"
)

// Runner applies migrations against one database.
type Runner struct {
	pool   *pgxpool.Pool
	db     *sql.DB
	goose  *goose.Provider
	logger *slog.Logger
}

// New opens a runner. The caller closes it.
func New(ctx context.Context, url string, logger *slog.Logger) (*Runner, error) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	// Migration guardrails (doc 10 §16).
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "3000"
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "300000"
	cfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sqlDB := stdlib.OpenDBFromPool(pool)
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		pool.Close()
		return nil, err
	}
	prov, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS, goose.WithSessionLocker(locker))
	if err != nil {
		_ = sqlDB.Close()
		pool.Close()
		return nil, fmt.Errorf("migrate: goose provider: %w", err)
	}
	return &Runner{pool: pool, db: sqlDB, goose: prov, logger: logger}, nil
}

// Close releases connections.
func (r *Runner) Close() {
	_ = r.goose.Close()
	r.pool.Close()
}

// Up applies River's migrations first (rovo's grants migration then covers river_* tables),
// then all pending goose migrations.
func (r *Runner) Up(ctx context.Context) error {
	rm, err := rivermigrate.New(riverpgxv5.New(r.pool), nil)
	if err != nil {
		return err
	}
	res, err := rm.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("migrate: river: %w", err)
	}
	for _, v := range res.Versions {
		r.logger.Info("river migration applied", "version", v.Version)
	}
	applied, err := r.goose.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: goose up: %w", err)
	}
	for _, a := range applied {
		r.logger.Info("migration applied", "version", a.Source.Version, "file", a.Source.Path, "duration", a.Duration)
	}
	return nil
}

// Down rolls back the latest goose migration (dev only; production rolls forward).
func (r *Runner) Down(ctx context.Context) error {
	res, err := r.goose.Down(ctx)
	if err != nil {
		return fmt.Errorf("migrate: goose down: %w", err)
	}
	if res != nil {
		r.logger.Info("migration rolled back", "version", res.Source.Version, "file", res.Source.Path)
	}
	return nil
}

// Reset rolls back every goose migration (tests).
func (r *Runner) Reset(ctx context.Context) error {
	_, err := r.goose.DownTo(ctx, 0)
	return err
}

// Status describes each goose migration.
type Status struct {
	Version int64
	File    string
	Applied bool
}

// Status lists goose migrations and whether they are applied.
func (r *Runner) Status(ctx context.Context) ([]Status, error) {
	st, err := r.goose.Status(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Status, 0, len(st))
	for _, s := range st {
		out = append(out, Status{Version: s.Source.Version, File: s.Source.Path, Applied: s.State == goose.StateApplied})
	}
	return out, nil
}

// Up is a convenience wrapper: open, migrate up, close.
func Up(ctx context.Context, url string, logger *slog.Logger) error {
	r, err := New(ctx, url, logger)
	if err != nil {
		return err
	}
	defer r.Close()
	return r.Up(ctx)
}
