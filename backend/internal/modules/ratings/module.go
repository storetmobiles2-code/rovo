// Package ratings owns restaurant star ratings with tags and optional review text (profanity
// filter, admin hide), rider thumbs up/down, the 7-day rating window and the rating aggregates
// ("New" until 5 ratings), per charter §3. Backend engineer C implements the service; this file
// fixes the public contract and the job-arg types.
package ratings

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/httpx"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/queue"
)

// Service is the ratings module's public API. Other modules reach it through
// app.Deps.Service(app.ServiceRatings).
type Service interface {
	// Submit records the customer's rating(s) for a delivered order inside the rating window.
	Submit(ctx context.Context, req SubmitRequest) error
	// Aggregate returns the public aggregate for a restaurant or rider.
	Aggregate(ctx context.Context, target TargetType, targetID uuid.UUID) (Aggregate, error)
	// ListReviews returns a restaurant's visible reviews, newest first.
	ListReviews(ctx context.Context, restaurantID uuid.UUID, cursor string, limit int) (ReviewPage, error)
	// HideReview hides a review (admin moderation) with a reason.
	HideReview(ctx context.Context, reviewID, adminID uuid.UUID, reason string) error
}

// TargetType is what a rating is about.
type TargetType string

// Rating targets.
const (
	TargetRestaurant TargetType = "RESTAURANT"
	TargetRider      TargetType = "RIDER"
)

// MinRatingsForDisplay is the count below which a restaurant shows "New" instead of a score.
const MinRatingsForDisplay = 5

// SubmitRequest is the rating input for one order. Stars (1–5) rate the restaurant; RiderThumbsUp
// is nil when the customer skipped the rider question.
type SubmitRequest struct {
	OrderID       uuid.UUID
	RaterUserID   uuid.UUID
	Stars         int
	Tags          []string
	ReviewText    string
	RiderThumbsUp *bool
}

// Aggregate is the public rating summary of a target.
type Aggregate struct {
	TargetType    TargetType
	TargetID      uuid.UUID
	RatingCount   int
	AverageStars  float64
	ThumbsUpCount int
	IsNew         bool
}

// Review is one restaurant review as shown publicly.
type Review struct {
	ID        uuid.UUID
	Stars     int
	Tags      []string
	Body      string
	CreatedAt string
}

// ReviewPage is a cursor page of reviews.
type ReviewPage struct {
	Items      []Review
	NextCursor string
}

// Event-type constants published by ratings.
const (
	EventRatingSubmitted = "ratings.rating_submitted.v1"
	EventReviewHidden    = "ratings.review_hidden.v1"
)

// PromptArgs nudges the customer to rate an order after ratings.prompt_after_s.
type PromptArgs struct {
	OrderID uuid.UUID `json:"orderId"`
}

// Kind implements queue.JobArgs.
func (PromptArgs) Kind() string { return "ratings.prompt" }

// RecomputeAggregateArgs recomputes a target's aggregate (30-day average) after a change.
type RecomputeAggregateArgs struct {
	TargetType string    `json:"targetType"`
	TargetID   uuid.UUID `json:"targetId"`
}

// Kind implements queue.JobArgs.
func (RecomputeAggregateArgs) Kind() string { return "ratings.recompute_aggregate" }

type service struct{ deps *app.Deps }

// New constructs the ratings module.
func New(deps *app.Deps) *app.Module {
	s := &service{deps: deps}
	return &app.Module{Name: app.ServiceRatings, Service: Service(s), Routes: s.registerRoutes, Jobs: s.registerJobs}
}

func (s *service) Submit(context.Context, SubmitRequest) error {
	return apperr.NotImplemented("ratings.Submit")
}

func (s *service) Aggregate(context.Context, TargetType, uuid.UUID) (Aggregate, error) {
	return Aggregate{}, apperr.NotImplemented("ratings.Aggregate")
}

func (s *service) ListReviews(context.Context, uuid.UUID, string, int) (ReviewPage, error) {
	return ReviewPage{}, apperr.NotImplemented("ratings.ListReviews")
}

func (s *service) HideReview(context.Context, uuid.UUID, uuid.UUID, string) error {
	return apperr.NotImplemented("ratings.HideReview")
}

func (s *service) registerRoutes(_ *httpx.Mux, _ *app.Deps)    {}
func (s *service) registerJobs(_ *queue.Registry, _ *app.Deps) {}
