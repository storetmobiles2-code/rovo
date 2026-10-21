package events

import (
	"context"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
)

// MarkProcessed records (handler, eventID) in processed_events and reports whether this is the
// first time. Handlers call it inside their own transaction and skip the work when already is
// false (at-least-once delivery, idempotent handlers — 08 §4.1 rule 3).
func MarkProcessed(ctx context.Context, q db.Querier, handler string, eventID uuid.UUID) (first bool, err error) {
	tag, err := q.Exec(ctx, `INSERT INTO processed_events (handler, event_id) VALUES ($1, $2)
		ON CONFLICT (handler, event_id) DO NOTHING`, handler, eventID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
