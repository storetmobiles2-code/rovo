// Package httpx holds the HTTP plumbing shared by every module: JSON decoding with limits,
// RFC 9457 problem rendering, the middleware chain (request id, recover, access log, security
// headers, CSRF/origin, audience-from-Host, auth, rate limit) and the idempotency-key store.
// Handlers are thin: decode → validate → authorise → call service → map errors (charter §4).
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/obs"
)

// DefaultMaxBody is the default request body limit (charter §4).
const DefaultMaxBody int64 = 1 << 20 // 1 MiB

// problemContentType is the RFC 9457 media type.
const problemContentType = "application/problem+json"

// JSON writes v as JSON with status and optional headers already set by the caller.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// NoContent writes 204.
func NoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// problemBody is the wire shape of an RFC 9457 problem (11 §1.8).
type problemBody struct {
	Type        string              `json:"type"`
	Title       string              `json:"title"`
	Status      int                 `json:"status"`
	Detail      string              `json:"detail,omitempty"`
	Instance    string              `json:"instance,omitempty"`
	Code        string              `json:"code"`
	TraceID     string              `json:"traceId"`
	Retryable   bool                `json:"retryable"`
	FieldErrors []apperr.FieldError `json:"fieldErrors"`
}

// Error renders err as problem+json. Non-apperr errors become 500 INTERNAL and are logged with
// the cause; apperr extensions are merged into the body (quote, diff, currentVersion, ...).
func Error(w http.ResponseWriter, r *http.Request, err error) {
	e, ok := apperr.As(err)
	if !ok {
		e = apperr.Internal(err)
	}
	ctx := r.Context()
	if e.Status >= 500 {
		obs.L(ctx, defaultLogger).Error("request failed", "code", e.Code, "status", e.Status, "error", err.Error())
	}
	body := map[string]any{
		"type":        "/problems/" + kebab(e.Code),
		"title":       title(e),
		"status":      e.Status,
		"code":        e.Code,
		"traceId":     firstNonEmpty(obs.TraceID(ctx), obs.RequestID(ctx)),
		"retryable":   e.Retryable,
		"fieldErrors": e.Fields,
	}
	if body["fieldErrors"] == nil || len(e.Fields) == 0 {
		body["fieldErrors"] = []apperr.FieldError{}
	}
	if e.Detail != "" {
		body["detail"] = e.Detail
	}
	if r.URL != nil {
		body["instance"] = r.URL.Path
	}
	for k, v := range e.Extensions {
		body[k] = v
	}
	h := w.Header()
	h.Set("Content-Type", problemContentType)
	if e.RetryAfterSeconds > 0 {
		h.Set("Retry-After", strconv.Itoa(e.RetryAfterSeconds))
	}
	w.WriteHeader(e.Status)
	_ = json.NewEncoder(w).Encode(body)
}

func title(e *apperr.Error) string {
	if e.Title != "" {
		return e.Title
	}
	return http.StatusText(e.Status)
}

// DecodeJSON reads a JSON body with a size limit and DisallowUnknownFields (charter §4). It maps
// decode failures to 400 VALIDATION_FAILED / 413-style errors.
func DecodeJSON[T any](w http.ResponseWriter, r *http.Request, maxBody int64) (T, error) {
	var out T
	if maxBody <= 0 {
		maxBody = DefaultMaxBody
	}
	ct := r.Header.Get("Content-Type")
	if ct != "" && !hasJSONContentType(ct) {
		return out, apperr.BadRequest(apperr.CodeUnsupportedMediaType, "Content-Type must be application/json.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return out, apperr.BadRequest(apperr.CodeBodyTooLarge, fmt.Sprintf("Request body exceeds %d bytes.", maxBody))
		}
		return out, apperr.Validation(apperr.FieldError{Field: "body", Code: "INVALID_JSON", Detail: decodeDetail(err)}).
			Wrap(err)
	}
	if dec.More() {
		return out, apperr.Validation(apperr.FieldError{Field: "body", Code: "TRAILING_DATA", Detail: "unexpected trailing JSON"})
	}
	return out, nil
}

func decodeDetail(err error) string {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return "malformed JSON"
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		return "wrong type for field " + ute.Field
	}
	return "invalid request body"
}

func hasJSONContentType(ct string) bool {
	for i := 0; i < len(ct); i++ {
		if ct[i] == ';' {
			ct = ct[:i]
			break
		}
	}
	return trimSpace(ct) == "application/json"
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func kebab(code string) string {
	b := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch {
		case c == '_':
			b = append(b, '-')
		case c >= 'A' && c <= 'Z':
			b = append(b, c+32)
		default:
			b = append(b, c)
		}
	}
	return string(b)
}

var _ = io.Discard
