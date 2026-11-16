// Package ratelimit implements Postgres-backed token/fixed-window counters shared across
// replicas (R21, 12 §2.2). Keys never contain raw phones or IPs — callers pass an HMAC/blind
// key. The limiter uses the injected clock so windows are deterministic in tests (R19).
package ratelimit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
)

// Limiter counts hits per (key, window) in rate_limit_buckets.
type Limiter struct {
	q     db.Querier
	clock clock.Clock
}

// New creates a limiter over q (pool or tx).
func New(q db.Querier, clk clock.Clock) *Limiter { return &Limiter{q: q, clock: clk} }

// Decision is the outcome of an Allow call.
type Decision struct {
	Allowed    bool
	Remaining  int
	Limit      int
	RetryAfter time.Duration // until the current window rolls over
	ResetAt    time.Time
}

// Allow increments the counter for key in the current fixed window of length window and reports
// whether the hit is within limit. The window start is floored to the app clock, so all replicas
// agree. Expired rows are swept by retention; this call also lets a lone writer cheaply reclaim
// its own key by keying on window_start.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	now := l.clock.Now()
	ws := now.Truncate(window)
	windowS := int(window / time.Second)
	if windowS < 1 {
		windowS = 1
	}
	expires := ws.Add(2 * window)
	var hits int
	err := l.q.QueryRow(ctx, `
		INSERT INTO rate_limit_buckets (bucket_key, window_start, window_s, hits, expires_at)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (bucket_key, window_start)
		DO UPDATE SET hits = rate_limit_buckets.hits + 1
		RETURNING hits`, key, ws, windowS, expires).Scan(&hits)
	if err != nil {
		return Decision{}, err
	}
	reset := ws.Add(window)
	d := Decision{
		Limit:     limit,
		Remaining: max(0, limit-hits),
		ResetAt:   reset,
	}
	if hits <= limit {
		d.Allowed = true
	} else {
		d.RetryAfter = reset.Sub(now)
		if d.RetryAfter < time.Second {
			d.RetryAfter = time.Second
		}
	}
	return d, nil
}

// Peek returns the current hit count for key in the current window without incrementing.
func (l *Limiter) Peek(ctx context.Context, key string, window time.Duration) (int, error) {
	ws := l.clock.Now().Truncate(window)
	var hits int
	err := l.q.QueryRow(ctx, `SELECT hits FROM rate_limit_buckets WHERE bucket_key = $1 AND window_start = $2`, key, ws).Scan(&hits)
	if db.IsNoRows(err) {
		return 0, nil
	}
	return hits, err
}

// Sweep deletes expired buckets (called by retention.sweep). Returns rows deleted.
func Sweep(ctx context.Context, q db.Querier, clk clock.Clock, batch int) (int64, error) {
	tag, err := q.Exec(ctx, `DELETE FROM rate_limit_buckets WHERE ctid IN (
		SELECT ctid FROM rate_limit_buckets WHERE expires_at < $1 LIMIT $2)`, clk.Now(), batch)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

var _ = pgx.ErrNoRows
