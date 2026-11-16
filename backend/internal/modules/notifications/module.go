// Package notifications owns the notification inbox, the SSE transport fan-out (via
// platform/realtime), fake SMS and Web Push adapters, and the restaurant alert ladder (doc 15,
// charter §3). Feature engineer C implements the service logic; this file fixes the public
// contract: the Service interface, the SMSSender / PushSender provider ports with fake
// implementations, the DTOs, and the event/job-arg types, plus the Register / RegisterJobs
// wiring convention (charter §5).
package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/testhooks"
)

// Service is the notifications module's public API (charter §5). Other modules reach it through
// app.Deps.Service(app.ServiceNotifications).
type Service interface {
	// Enqueue records an inbox notification and fans it out over the enabled channels.
	Enqueue(ctx context.Context, n Notification) error
	// ListInbox returns a user's notification inbox page.
	ListInbox(ctx context.Context, userID uuid.UUID, cursor string, limit int) (InboxPage, error)
}

// Notification is one inbox/notification-channel message (doc 15 §2).
type Notification struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Category  string
	Title     string
	Body      string
	DeepLink  string
	Channels  []Channel
	CreatedAt time.Time
}

// Channel is a delivery channel.
type Channel string

// Channels.
const (
	ChannelInbox Channel = "INBOX"
	ChannelSMS   Channel = "SMS"
	ChannelPush  Channel = "PUSH"
)

// InboxPage is a cursor page of notifications.
type InboxPage struct {
	Items       []Notification
	NextCursor  string
	UnreadCount int
}

// SMSSender is the SMS provider port (doc 15 §4). The fake writes to the testhooks outbox; the
// real DLT adapter lands beside it in a later slice.
type SMSSender interface {
	SendSMS(ctx context.Context, toE164, templateID string, vars map[string]string) (providerMsgID string, err error)
}

// PushSender is the Web Push (VAPID) provider port (doc 15 §4.3).
type PushSender interface {
	SendPush(ctx context.Context, subscription PushSubscription, payload PushPayload) error
}

// PushSubscription is a W3C Push subscription record.
type PushSubscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// PushPayload is the encrypted, minimal push body (never PII; doc 15 §4.3).
type PushPayload struct {
	Type     string `json:"t"`
	EntityID string `json:"id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	DeepLink string `json:"deepLink"`
}

// Job-arg types (River). Each subscriber/timer gets a typed arg implementing Kind().

// RestaurantAlertArgs re-rings a restaurant until the order is acknowledged (R1, doc 15).
type RestaurantAlertArgs struct {
	OrderID      uuid.UUID `json:"orderId"`
	RestaurantID uuid.UUID `json:"restaurantId"`
	Attempt      int       `json:"attempt"`
}

// Kind implements queue.JobArgs.
func (RestaurantAlertArgs) Kind() string { return "notifications.restaurant_alert" }

type service struct {
	deps *app.Deps
	sms  SMSSender
	push PushSender
}

// New constructs the notifications module with the configured providers.
func New(deps *app.Deps, sms SMSSender, push PushSender) *app.Module {
	s := &service{deps: deps, sms: sms, push: push}
	return &app.Module{
		Name:    app.ServiceNotifications,
		Service: Service(s),
		Routes:  s.registerRoutes,
		Jobs:    s.registerJobs,
	}
}

// Enqueue is a stub; feature engineer C implements inbox persistence and fan-out.
func (s *service) Enqueue(context.Context, Notification) error {
	return apperr.NotImplemented("notifications.Enqueue")
}

// ListInbox is a stub.
func (s *service) ListInbox(context.Context, uuid.UUID, string, int) (InboxPage, error) {
	return InboxPage{}, apperr.NotImplemented("notifications.ListInbox")
}

// registerRoutes wires the notification HTTP surface. The SSE transport (GET /stream) is mounted
// by the composition root because its lifecycle is tied to the server drain context; the inbox
// endpoints land here when the service is implemented.
func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps) {}

// registerJobs wires the restaurant alert ladder worker (stub registration).
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}

// FakeSMS writes SMS to the process-local testhooks outbox so E2E tests can read them (QA-16).
type FakeSMS struct{ Outbox *testhooks.Outbox }

// SendSMS records the message and returns a synthetic provider id.
func (f FakeSMS) SendSMS(_ context.Context, toE164, templateID string, vars map[string]string) (string, error) {
	if f.Outbox != nil {
		f.Outbox.Add(testhooks.Message{Channel: "SMS", To: toE164, Body: templateID, Meta: vars, CreatedAt: time.Now().UTC()})
	}
	return "fake-sms-" + uuid.NewString(), nil
}

var _ SMSSender = FakeSMS{}

// FakePush records Web Push sends to the testhooks outbox.
type FakePush struct{ Outbox *testhooks.Outbox }

// SendPush records the payload.
func (f FakePush) SendPush(_ context.Context, sub PushSubscription, payload PushPayload) error {
	if f.Outbox != nil {
		f.Outbox.Add(testhooks.Message{Channel: "PUSH", To: sub.Endpoint, Body: payload.Title, CreatedAt: time.Now().UTC()})
	}
	return nil
}

var _ PushSender = FakePush{}
