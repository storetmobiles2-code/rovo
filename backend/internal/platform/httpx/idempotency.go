package httpx

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/db"
)

// IdempotencyStore backs the Idempotency-Key semantics (08 §7.1) in idempotency_keys. The
// handler's business transaction writes the COMPLETED row so the key and the effect commit
// atomically; this store handles the IN_PROGRESS lease and replay.
type IdempotencyStore struct {
	clock clock.Clock
}

// NewIdempotencyStore builds a store.
func NewIdempotencyStore(clk clock.Clock) *IdempotencyStore { return &IdempotencyStore{clock: clk} }

// CompletedResponse is a stored idempotent response, replayed on a repeat request.
type CompletedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

// RequestHash hashes method|route|body for same-key/different-body detection.
func RequestHash(method, route string, body []byte) []byte {
	h := sha256.New()
	h.Write([]byte(method))
	h.Write([]byte{0})
	h.Write([]byte(route))
	h.Write([]byte{0})
	h.Write(body)
	return h.Sum(nil)
}

// PrincipalID returns the idempotency principal id: the user id, or anon:<ip-hmac> fallback.
func PrincipalID(p *auth.Principal, anonFallback string) string {
	if p != nil && p.UserID != uuid.Nil {
		return p.UserID.String()
	}
	return "anon:" + anonFallback
}

// Begin claims the key against q (pool or tx). It returns a non-nil *CompletedResponse to replay
// a prior COMPLETED result; apperr 422 IDEMPOTENCY_KEY_REUSED for a key reused with a different
// body; apperr 409 REQUEST_IN_PROGRESS while another request holds the lease. A fresh claim
// returns (nil, nil); the handler then calls Complete inside its business tx.
func (s *IdempotencyStore) Begin(ctx context.Context, q db.Querier, principalID, key, operationID string, reqHash []byte) (*CompletedResponse, error) {
	now := s.clock.Now()
	tag, err := q.Exec(ctx, `INSERT INTO idempotency_keys
		(principal_id, key, operation_id, request_hash, status, locked_until, created_at, expires_at)
		VALUES ($1,$2,$3,$4,'IN_PROGRESS',$5,$6,$7)
		ON CONFLICT (principal_id, key) DO NOTHING`,
		principalID, key, operationID, reqHash, now.Add(30*time.Second), now, now.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 1 {
		return nil, nil // fresh claim
	}
	var (
		status     string
		storedHash []byte
		respStatus *int16
		respBody   []byte
		respHdr    []byte
		lockedTill *time.Time
	)
	err = q.QueryRow(ctx, `SELECT status, request_hash, response_status, response_body, response_headers, locked_until
		FROM idempotency_keys WHERE principal_id=$1 AND key=$2`, principalID, key).
		Scan(&status, &storedHash, &respStatus, &respBody, &respHdr, &lockedTill)
	if err != nil {
		return nil, err
	}
	if !equalBytes(storedHash, reqHash) {
		return nil, apperr.Unprocessable(apperr.CodeIdempotencyKeyReused,
			"This Idempotency-Key was used with a different request.")
	}
	if status == "COMPLETED" && respStatus != nil {
		cr := &CompletedResponse{Status: int(*respStatus), Headers: map[string]string{}, Body: respBody}
		_ = json.Unmarshal(respHdr, &cr.Headers)
		return cr, nil
	}
	if lockedTill != nil && now.After(*lockedTill) {
		return nil, nil // stale lease; reclaim
	}
	return nil, apperr.Conflict(apperr.CodeRequestInProgress, "An identical request is already in progress.").
		With("retryAfterSeconds", 1)
}

// Complete records the response for replay, inside the handler's tx (q is usually the tx).
func (s *IdempotencyStore) Complete(ctx context.Context, q db.Querier, principalID, key string, cr CompletedResponse) error {
	hdr, _ := json.Marshal(cr.Headers)
	_, err := q.Exec(ctx, `UPDATE idempotency_keys
		SET status='COMPLETED', response_status=$3, response_body=$4, response_headers=$5, completed_at=$6
		WHERE principal_id=$1 AND key=$2`,
		principalID, key, int16(cr.Status), cr.Body, hdr, s.clock.Now())
	return err
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
