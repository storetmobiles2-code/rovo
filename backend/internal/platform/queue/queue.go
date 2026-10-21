// Package queue wraps riverqueue so modules never import River directly (08 §4.1 rule 5). It is
// the transactional outbox (R22/R42): publishers insert one job per subscriber with InsertManyTx
// in the business transaction. Modules register workers and periodic jobs at wiring time, then
// the worker process starts the client.
package queue

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

// Queue names (08 §7.3): realtime for latency-critical fan-out, default for the rest, batch for
// heavy/retention work.
const (
	QueueRealtime = "realtime"
	QueueDefault  = "default"
	QueueBatch    = "batch"
)

// JobArgs is any River job argument type (implements Kind()). Re-exported so modules depend on
// platform/queue, not River.
type JobArgs = river.JobArgs

// Job is a unit of work passed to a worker's Work method.
type Job[T JobArgs] = river.Job[T]

// InsertOpts controls scheduling, queue, uniqueness and retries.
type InsertOpts struct {
	Queue       string
	ScheduledAt time.Time     // zero = now
	MaxAttempts int           // 0 = River default
	Priority    int           // 1 (highest) .. 4
	UniqueByArgs bool         // dedupe identical args (idempotent timers/subscribers)
	UniquePeriod time.Duration
}

func (o InsertOpts) river() *river.InsertOpts {
	ro := &river.InsertOpts{Queue: o.Queue, ScheduledAt: o.ScheduledAt, MaxAttempts: o.MaxAttempts, Priority: o.Priority}
	if o.UniqueByArgs {
		ro.UniqueOpts = river.UniqueOpts{ByArgs: true, ByPeriod: o.UniquePeriod}
	}
	return ro
}

// InsertParam pairs args with their opts for a batched insert.
type InsertParam struct {
	Args JobArgs
	Opts InsertOpts
}

// Registry collects workers and periodic jobs during wiring. Each module calls RegisterJobs
// with it. It is sealed when the client is built.
type Registry struct {
	workers  *river.Workers
	periodic []*river.PeriodicJob
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry { return &Registry{workers: river.NewWorkers()} }

// Workers exposes the underlying *river.Workers so modules can call
// river.AddWorker(reg.Workers(), w) with their typed workers.
func (r *Registry) Workers() *river.Workers { return r.workers }

// AddPeriodic registers a catch-up periodic job (13 §5.2). schedule is a cron/interval; the
// constructor returns the args to insert each tick.
func (r *Registry) AddPeriodic(every time.Duration, args JobArgs, opts InsertOpts) {
	r.periodic = append(r.periodic, river.NewPeriodicJob(
		river.PeriodicInterval(every),
		func() (river.JobArgs, *river.InsertOpts) { return args, opts.river() },
		&river.PeriodicJobOpts{RunOnStart: true},
	))
}

// Client is the rovo job client over a pgx pool.
type Client struct {
	rc   *river.Client[pgx.Tx]
	pool *pgxpool.Pool
}

// Config configures the worker client.
type Config struct {
	// Queues maps queue name → max concurrent workers. Nil means insert-only (the api process).
	Queues map[string]int
}

// New builds a client. With cfg.Queues nil it is insert-only (Start is a no-op); the worker
// process passes queue concurrencies to actually work jobs.
func New(pool *pgxpool.Pool, reg *Registry, cfg Config) (*Client, error) {
	rcfg := &river.Config{Workers: reg.workers, PeriodicJobs: reg.periodic}
	if len(cfg.Queues) > 0 {
		rcfg.Queues = map[string]river.QueueConfig{}
		for name, n := range cfg.Queues {
			rcfg.Queues[name] = river.QueueConfig{MaxWorkers: n}
		}
	}
	rc, err := river.NewClient(riverpgxv5.New(pool), rcfg)
	if err != nil {
		return nil, err
	}
	return &Client{rc: rc, pool: pool}, nil
}

// Start begins working jobs (worker process only).
func (c *Client) Start(ctx context.Context) error { return c.rc.Start(ctx) }

// Stop drains and stops the worker gracefully.
func (c *Client) Stop(ctx context.Context) error {
	if c == nil || c.rc == nil {
		return nil
	}
	return c.rc.Stop(ctx)
}

// Enqueue inserts a job outside a transaction (timers rescheduled from a worker, best-effort).
func (c *Client) Enqueue(ctx context.Context, args JobArgs, opts InsertOpts) error {
	_, err := c.rc.Insert(ctx, args, opts.river())
	return err
}

// EnqueueTx inserts a job in tx (the business transaction) — the outbox write.
func (c *Client) EnqueueTx(ctx context.Context, tx pgx.Tx, args JobArgs, opts InsertOpts) error {
	_, err := c.rc.InsertTx(ctx, tx, args, opts.river())
	return err
}

// EnqueueManyTx inserts one job per subscriber in tx (InsertManyTx, R42).
func (c *Client) EnqueueManyTx(ctx context.Context, tx pgx.Tx, params []InsertParam) error {
	if len(params) == 0 {
		return nil
	}
	many := make([]river.InsertManyParams, len(params))
	for i, p := range params {
		many[i] = river.InsertManyParams{Args: p.Args, InsertOpts: p.Opts.river()}
	}
	_, err := c.rc.InsertManyTx(ctx, tx, many)
	return err
}

// Enqueuer is the narrow interface modules receive (publishers, timer schedulers). It keeps
// River out of module code.
type Enqueuer interface {
	Enqueue(ctx context.Context, args JobArgs, opts InsertOpts) error
	EnqueueTx(ctx context.Context, tx pgx.Tx, args JobArgs, opts InsertOpts) error
	EnqueueManyTx(ctx context.Context, tx pgx.Tx, params []InsertParam) error
}

var _ Enqueuer = (*Client)(nil)

// ErrNoClient is returned by helpers when the client was not built.
var ErrNoClient = errors.New("queue: client not configured")

// WorkerDefaults embeds River's worker defaults; module workers embed it.
type WorkerDefaults[T JobArgs] = river.WorkerDefaults[T]

// AddWorker registers a typed worker on the registry.
func AddWorker[T JobArgs](r *Registry, w river.Worker[T]) { river.AddWorker(r.workers, w) }

var _ rivertype.JobInsertResult
