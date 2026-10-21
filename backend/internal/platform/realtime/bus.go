// Package realtime implements the SSE delivery path: a Postgres LISTEN/NOTIFY bus on channel
// rovo_events (transactional hints, 08 §6.2) and an in-process hub that fans frames out to the
// SSE connections on this replica. There is no replay buffer; clients refetch over REST (R10).
package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Channel is the Postgres NOTIFY channel.
const Channel = "rovo_events"

// Frame is the thin envelope sent over NOTIFY and to SSE clients (≤ 8000 bytes on the wire).
type Frame struct {
	Topic   string          `json:"topic"`
	Event   string          `json:"event"`   // SSE event: name (11 §4.2)
	ID      string          `json:"id"`      // <entityType>:<entityId>:<version>
	Data    json.RawMessage `json:"data"`
}

// Bus publishes frames (transactionally via NotifyTx) and, on the subscriber side, reads the
// LISTEN channel and dispatches to the local hub.
type Bus struct {
	pool   *pgxpool.Pool
	hub    *Hub
	logger *slog.Logger
	onState func(up bool)
}

// NewBus builds a bus over pool with a fresh hub.
func NewBus(pool *pgxpool.Pool, logger *slog.Logger) *Bus {
	return &Bus{pool: pool, hub: NewHub(), logger: logger}
}

// Hub exposes the in-process hub for the SSE handler.
func (b *Bus) Hub() *Hub { return b.hub }

// OnStateChange registers a callback for LISTEN connectivity (health readiness).
func (b *Bus) OnStateChange(fn func(up bool)) { b.onState = fn }

// NotifyTx emits a frame for topic inside tx. Delivery happens only if tx commits, giving outbox
// semantics for the hint (08 §6.2). event/id can be derived by the caller; Publish-style callers
// pass a full Frame via NotifyFrameTx.
func (b *Bus) NotifyTx(ctx context.Context, tx pgx.Tx, topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return b.NotifyFrameTx(ctx, tx, Frame{Topic: topic, Data: data})
}

// NotifyFrameTx emits a full frame inside tx.
func (b *Bus) NotifyFrameTx(ctx context.Context, tx pgx.Tx, f Frame) error {
	b.hub.capFrame(&f)
	payload, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if len(payload) > 7800 { // NOTIFY payload limit is 8000 bytes
		// Drop the data; clients refetch over REST anyway (R10).
		f.Data = json.RawMessage(`{}`)
		if payload, err = json.Marshal(f); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `SELECT pg_notify($1, $2)`, Channel, string(payload))
	return err
}

// Run opens a dedicated LISTEN connection and dispatches frames to the hub until ctx is done. It
// reconnects with backoff; a watchdog probe (self-NOTIFY every 10 s) detects a wedged connection
// (M12). On each state change it calls the OnStateChange callback.
func (b *Bus) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := b.listen(ctx); err != nil && ctx.Err() == nil {
			b.setState(false)
			b.logger.Warn("realtime listen connection lost", "error", err, "retry", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (b *Bus) listen(ctx context.Context) error {
	conn, err := b.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	b.setState(true)
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var f Frame
		if err := json.Unmarshal([]byte(n.Payload), &f); err != nil {
			b.logger.Warn("realtime bad frame", "error", err)
			continue
		}
		b.hub.Dispatch(f)
	}
}

func (b *Bus) setState(up bool) {
	if b.onState != nil {
		b.onState(up)
	}
}

// ErrBusClosed is returned by a closed hub.
var ErrBusClosed = errors.New("realtime: hub closed")

var _ = pgxpool.Pool{}
var _ sync.Mutex
