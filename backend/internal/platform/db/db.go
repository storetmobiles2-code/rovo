// Package db wraps pgx: the connection pool, transactions and small error helpers. Every
// query in rovo is hand-written, parameterised SQL (ADR-P2-002); this package never builds SQL
// from user input.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is satisfied by *pgxpool.Pool, *pgxpool.Conn and pgx.Tx, so store functions accept
// either a pool or the caller's transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Options tune the pool. Zero values use the defaults below.
type Options struct {
	MaxConns          int32
	StatementTimeout  time.Duration // 5 s for api, 60 s for jobs (08 §7.2)
	IdleInTxTimeout   time.Duration
	ApplicationName   string
}

// NewPool opens a pgx pool and pings it.
func NewPool(ctx context.Context, url string, o Options) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("db: parse url: %w", err)
	}
	if o.MaxConns > 0 {
		cfg.MaxConns = o.MaxConns
	}
	if o.StatementTimeout == 0 {
		o.StatementTimeout = 5 * time.Second
	}
	if o.IdleInTxTimeout == 0 {
		o.IdleInTxTimeout = 10 * time.Second
	}
	rp := cfg.ConnConfig.RuntimeParams
	rp["statement_timeout"] = fmt.Sprint(o.StatementTimeout.Milliseconds())
	rp["idle_in_transaction_session_timeout"] = fmt.Sprint(o.IdleInTxTimeout.Milliseconds())
	rp["timezone"] = "UTC"
	if o.ApplicationName != "" {
		rp["application_name"] = o.ApplicationName
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: open pool: %w", err)
	}
	pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return pool, nil
}

// TxBeginner is implemented by *pgxpool.Pool.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// WithTx runs fn in a READ COMMITTED transaction, committing on nil and rolling back on error
// or panic. Deferred constraint triggers (ledger balance, order bill) fire at commit, so a
// commit error is returned as-is.
func WithTx(ctx context.Context, b TxBeginner, fn func(tx pgx.Tx) error) (err error) {
	return WithTxOptions(ctx, b, pgx.TxOptions{}, fn)
}

// WithTxOptions is WithTx with explicit options (e.g. Serializable for settlement runs).
func WithTxOptions(ctx context.Context, b TxBeginner, opts pgx.TxOptions, fn func(tx pgx.Tx) error) (err error) {
	tx, err := b.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("db: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// InTx is WithTx returning a value.
func InTx[T any](ctx context.Context, b TxBeginner, fn func(tx pgx.Tx) (T, error)) (T, error) {
	var out T
	err := WithTx(ctx, b, func(tx pgx.Tx) error {
		var e error
		out, e = fn(tx)
		return e
	})
	return out, err
}

// SQLSTATE codes used by callers.
const (
	SQLStateUniqueViolation     = "23505"
	SQLStateForeignKeyViolation = "23503"
	SQLStateCheckViolation      = "23514"
	SQLStateExclusionViolation  = "23P01"
	SQLStateSerialization       = "40001"
	SQLStateInsufficientPriv    = "42501"
)

// PgError returns the *pgconn.PgError in err, if any.
func PgError(err error) (*pgconn.PgError, bool) {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe, true
	}
	return nil, false
}

// IsUniqueViolation reports a unique violation, optionally on a specific constraint/index name.
func IsUniqueViolation(err error, constraint ...string) bool {
	return isCode(err, SQLStateUniqueViolation, constraint...)
}

// IsExclusionViolation reports an EXCLUDE constraint violation (overlapping ranges).
func IsExclusionViolation(err error, constraint ...string) bool {
	return isCode(err, SQLStateExclusionViolation, constraint...)
}

// IsCheckViolation reports a CHECK violation (also raised by rovo's validation triggers).
func IsCheckViolation(err error, constraint ...string) bool {
	return isCode(err, SQLStateCheckViolation, constraint...)
}

// IsNoRows reports pgx.ErrNoRows.
func IsNoRows(err error) bool { return errors.Is(err, pgx.ErrNoRows) }

func isCode(err error, code string, constraint ...string) bool {
	pe, ok := PgError(err)
	if !ok || pe.Code != code {
		return false
	}
	if len(constraint) == 0 {
		return true
	}
	for _, c := range constraint {
		if pe.ConstraintName == c {
			return true
		}
	}
	return false
}

// Connect opens a single dedicated connection (LISTEN, migrations). It is outside the pool so
// LISTEN keeps working when the pool sits behind a transaction-mode pooler.
func Connect(ctx context.Context, url string) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	return pgx.ConnectConfig(ctx, cfg)
}
