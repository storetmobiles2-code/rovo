package app

import (
	"context"
	"time"

	platform "github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/ratelimit"
)

// RateLimitSweepArgs deletes expired rate_limit_buckets rows (platform housekeeping).
type RateLimitSweepArgs struct{}

// Kind implements queue.JobArgs.
func (RateLimitSweepArgs) Kind() string { return "platform.ratelimit_sweep" }

type rateLimitSweepWorker struct {
	queue.WorkerDefaults[RateLimitSweepArgs]
	deps *platform.Deps
}

func (w *rateLimitSweepWorker) Work(ctx context.Context, _ *queue.Job[RateLimitSweepArgs]) error {
	n, err := ratelimit.Sweep(ctx, w.deps.Pool, w.deps.Clock, 5000)
	if err != nil {
		return err
	}
	if n > 0 {
		w.deps.Logger.Info("rate-limit buckets swept", "deleted", n)
	}
	return nil
}

// registerPlatformJobs wires the jobs owned by the platform rather than a domain module. It also
// guarantees the worker process always has at least one River worker, which River requires.
func registerPlatformJobs(reg *queue.Registry, deps *platform.Deps) {
	queue.AddWorker(reg, &rateLimitSweepWorker{deps: deps})
	reg.AddPeriodic(10*time.Minute, RateLimitSweepArgs{}, queue.InsertOpts{
		Queue: queue.QueueDefault, UniqueByArgs: true, UniquePeriod: 10 * time.Minute,
	})
}
