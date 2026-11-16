// Package support owns support tickets and their message threads, the undeliverable-request
// approval flow (rider request → support decision), compensation decisions (refund or goodwill
// coupon, R29) and ticket SLAs (doc 10 §10, charter §3). Backend engineer C implements the
// service; this file fixes the public contract and the job-arg types.
package support

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the support module's public API. Other modules reach it through
// app.Deps.Service(app.ServiceSupport).
type Service interface {
	// Open creates a ticket; SYSTEM-opened tickets (e.g. an undeliverable request) pass a zero
	// RequesterUserID.
	Open(ctx context.Context, req OpenRequest) (Ticket, error)
	// Get returns a ticket the caller may view.
	Get(ctx context.Context, ticketID uuid.UUID) (Ticket, error)
	// AddMessage appends a message or internal note to a ticket thread.
	AddMessage(ctx context.Context, m Message) error
	// Resolve closes the loop with a resolution code (reason_codes TICKET_RESOLUTION).
	Resolve(ctx context.Context, r Resolution) (Ticket, error)
	// DecideUndeliverable approves or rejects a rider's undeliverable request.
	DecideUndeliverable(ctx context.Context, ticketID, adminID uuid.UUID, approve bool, note string) error
}

// Category is a ticket category (support_tickets.category).
type Category string

// Ticket categories used by other modules; the full list is in the migration CHECK.
const (
	CategoryMissingItems           Category = "MISSING_ITEMS"
	CategoryWrongItems             Category = "WRONG_ITEMS"
	CategoryQuality                Category = "QUALITY"
	CategoryLateDelivery           Category = "LATE_DELIVERY"
	CategoryNotDelivered           Category = "NOT_DELIVERED"
	CategoryPaymentIssue           Category = "PAYMENT_ISSUE"
	CategoryRestaurantCannotFulfil Category = "RESTAURANT_CANNOT_FULFIL"
	CategoryUndeliverableRequest   Category = "UNDELIVERABLE_REQUEST"
	CategoryOther                  Category = "OTHER"
)

// Status is a ticket status.
type Status string

// Ticket statuses.
const (
	StatusOpen              Status = "OPEN"
	StatusInProgress        Status = "IN_PROGRESS"
	StatusAwaitingRequester Status = "AWAITING_REQUESTER"
	StatusAwaitingApproval  Status = "AWAITING_APPROVAL"
	StatusResolved          Status = "RESOLVED"
	StatusClosed            Status = "CLOSED"
	StatusReopened          Status = "REOPENED"
)

// OpenRequest is the ticket-creation input.
type OpenRequest struct {
	CityID          uuid.UUID
	RequesterUserID uuid.UUID
	RequesterApp    string
	OrderID         uuid.UUID
	Category        Category
	Subject         string
	Body            string
	Priority        string
}

// Ticket is the ticket view.
type Ticket struct {
	ID        uuid.UUID
	Code      string
	Category  Category
	Status    Status
	Priority  string
	Subject   string
	OrderID   uuid.UUID
	SLADueAt  time.Time
	Version   int
	CreatedAt time.Time
}

// Message is one thread entry. Internal notes are never shown to the requester.
type Message struct {
	TicketID     uuid.UUID
	AuthorUserID uuid.UUID
	Body         string
	Internal     bool
}

// Resolution is the support agent's outcome for a ticket.
type Resolution struct {
	TicketID          uuid.UUID
	AdminID           uuid.UUID
	ResolutionCode    string
	Note              string
	FaultParty        string
	CompensationPaise int64
	Version           int
}

// Event-type constants published by support.
const (
	EventTicketOpened         = "support.ticket_opened.v1"
	EventTicketResolved       = "support.ticket_resolved.v1"
	EventUndeliverableDecided = "support.undeliverable_decided.v1"
)

// SLACheckArgs escalates a ticket whose SLA is breached.
type SLACheckArgs struct {
	TicketID uuid.UUID `json:"ticketId"`
}

// Kind implements queue.JobArgs.
func (SLACheckArgs) Kind() string { return "support.sla_check" }

type service struct{ deps *app.Deps }

// New constructs the support module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceSupport, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Open(context.Context, OpenRequest) (Ticket, error) {
	return Ticket{}, apperr.NotImplemented("support.Open")
}

func (s *service) Get(context.Context, uuid.UUID) (Ticket, error) {
	return Ticket{}, apperr.NotImplemented("support.Get")
}

func (s *service) AddMessage(context.Context, Message) error {
	return apperr.NotImplemented("support.AddMessage")
}

func (s *service) Resolve(context.Context, Resolution) (Ticket, error) {
	return Ticket{}, apperr.NotImplemented("support.Resolve")
}

func (s *service) DecideUndeliverable(context.Context, uuid.UUID, uuid.UUID, bool, string) error {
	return apperr.NotImplemented("support.DecideUndeliverable")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
