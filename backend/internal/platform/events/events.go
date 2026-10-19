// Package events defines the domain-event envelope (08 §3.3) and the Publish helper. There is no
// event table (R42): Publish looks up the static subscription table and inserts one River job per
// subscriber in the caller's transaction (queue.EnqueueManyTx). Handlers dedupe on
// (handler, event_id) in processed_events.
package events

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Envelope is the common header carried in every subscriber job's args. Payloads carry ids and
// minimal facts; consumers that need more call the owning module's reader (08 §3.3).
type Envelope struct {
	ID               uuid.UUID `json:"id"`
	Type             string    `json:"type"`    // e.g. "ordering.order_accepted.v1"
	Name             string    `json:"name"`    // e.g. "OrderAccepted"
	Version          int       `json:"version"`
	OccurredAt       time.Time `json:"occurredAt"`
	CityID           uuid.UUID `json:"cityId"`
	AggregateType    string    `json:"aggregateType"`
	AggregateID      uuid.UUID `json:"aggregateId"`
	AggregateVersion int       `json:"aggregateVersion"`
	Actor            Actor     `json:"actor"`
	TraceParent      string    `json:"traceparent,omitempty"`
}

// Actor is who caused the event.
type Actor struct {
	Type string     `json:"type"`
	ID   *uuid.UUID `json:"id,omitempty"`
	Role string     `json:"role,omitempty"`
}

// Subscriber turns one event into one River job. Kind is the River job kind
// (<module>.on_<event>); ArgsFor builds the job args from the envelope and raw payload.
type Subscriber struct {
	Handler string // processed_events key and River kind
	Queue   string
	Build   func(env Envelope, payload []byte) queue.JobArgs
}

// Registry maps an event type to its subscribers. It is code, not config (R42): modules add
// their subscriptions at wiring time.
type Registry struct {
	subs map[string][]Subscriber
}

// NewRegistry returns an empty subscription registry.
func NewRegistry() *Registry { return &Registry{subs: map[string][]Subscriber{}} }

// Subscribe registers sub for eventType.
func (r *Registry) Subscribe(eventType string, sub Subscriber) {
	r.subs[eventType] = append(r.subs[eventType], sub)
}

// Subscribers returns the subscribers for eventType.
func (r *Registry) Subscribers(eventType string) []Subscriber { return r.subs[eventType] }

// Publisher inserts subscriber jobs transactionally and emits the realtime NOTIFY hint.
type Publisher struct {
	reg   *Registry
	queue queue.Enqueuer
	bus   Notifier
}

// Notifier publishes a realtime hint on commit (realtime.Bus implements it).
type Notifier interface {
	NotifyTx(ctx context.Context, tx pgx.Tx, topic string, payload any) error
}

// NewPublisher builds a publisher.
func NewPublisher(reg *Registry, q queue.Enqueuer, bus Notifier) *Publisher {
	return &Publisher{reg: reg, queue: q, bus: bus}
}

// Publish inserts one job per subscriber for env (with payload JSON) in tx. It is the only
// fan-out path. Realtime hints are emitted separately by the caller via PublishRealtime, which
// keeps this method payload-shape agnostic.
func (p *Publisher) Publish(ctx context.Context, tx pgx.Tx, env Envelope, payload []byte) error {
	subs := p.reg.Subscribers(env.Type)
	if len(subs) == 0 {
		return nil
	}
	params := make([]queue.InsertParam, 0, len(subs))
	for _, s := range subs {
		q := s.Queue
		if q == "" {
			q = queue.QueueDefault
		}
		params = append(params, queue.InsertParam{
			Args: s.Build(env, payload),
			Opts: queue.InsertOpts{Queue: q, UniqueByArgs: true, UniquePeriod: 24 * time.Hour},
		})
	}
	return p.queue.EnqueueManyTx(ctx, tx, params)
}

// PublishRealtime emits an SSE hint on topic in tx (transactional NOTIFY).
func (p *Publisher) PublishRealtime(ctx context.Context, tx pgx.Tx, topic string, payload any) error {
	if p.bus == nil {
		return nil
	}
	return p.bus.NotifyTx(ctx, tx, topic, payload)
}
