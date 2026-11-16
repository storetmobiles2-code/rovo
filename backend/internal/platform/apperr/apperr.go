// Package apperr defines the error type that services return and httpx renders as RFC 9457
// problem+json. Codes are the stable catalogue of 11 §1.8; clients branch on Code only.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// FieldError points at one invalid request field ("lines[0].quantity").
type FieldError struct {
	Field  string `json:"field"`
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}

// Error is a domain or request error with an HTTP status and a stable code.
type Error struct {
	Status    int
	Code      string
	Title     string
	Detail    string
	Fields    []FieldError
	Retryable bool
	// Extensions are extra problem members (quote, diff, currentStatus, currentVersion,
	// approvalRequest, retryAfterSeconds, attemptsRemaining...).
	Extensions map[string]any
	// RetryAfterSeconds sets the Retry-After header when > 0.
	RetryAfterSeconds int
	cause             error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Detail, e.cause)
	}
	if e.Detail != "" {
		return e.Code + ": " + e.Detail
	}
	return e.Code
}

// Unwrap exposes the underlying cause for errors.Is/As.
func (e *Error) Unwrap() error { return e.cause }

// With returns a copy carrying an extension member.
func (e *Error) With(key string, value any) *Error {
	c := *e
	c.Extensions = make(map[string]any, len(e.Extensions)+1)
	for k, v := range e.Extensions {
		c.Extensions[k] = v
	}
	c.Extensions[key] = value
	return &c
}

// Wrap attaches a cause (logged, never rendered).
func (e *Error) Wrap(cause error) *Error {
	c := *e
	c.cause = cause
	return &c
}

// New builds an error.
func New(status int, code, detail string) *Error {
	return &Error{Status: status, Code: code, Title: http.StatusText(status), Detail: detail}
}

// As extracts an *Error from err.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsCode reports whether err is an *Error with the given code.
func IsCode(err error, code string) bool {
	e, ok := As(err)
	return ok && e.Code == code
}

// Validation returns 400 VALIDATION_FAILED with field errors.
func Validation(fields ...FieldError) *Error {
	e := New(http.StatusBadRequest, CodeValidationFailed, "The request is invalid.")
	e.Fields = fields
	return e
}

// BadRequest returns a 400 with a specific code.
func BadRequest(code, detail string) *Error { return New(http.StatusBadRequest, code, detail) }

// Unauthenticated returns 401.
func Unauthenticated(code, detail string) *Error {
	if code == "" {
		code = CodeUnauthenticated
	}
	return New(http.StatusUnauthorized, code, detail)
}

// Forbidden returns 403.
func Forbidden(code, detail string) *Error {
	if code == "" {
		code = CodeForbidden
	}
	return New(http.StatusForbidden, code, detail)
}

// NotFound returns 404 NOT_FOUND; also used for resources outside the caller's scope.
func NotFound(detail string) *Error { return New(http.StatusNotFound, CodeNotFound, detail) }

// Conflict returns 409 with a specific code.
func Conflict(code, detail string) *Error { return New(http.StatusConflict, code, detail) }

// PreconditionFailed returns 412 VERSION_MISMATCH with currentVersion.
func PreconditionFailed(currentVersion int) *Error {
	return New(http.StatusPreconditionFailed, CodeVersionMismatch, "The resource was changed by someone else.").
		With("currentVersion", currentVersion)
}

// PreconditionRequired returns 428 IF_MATCH_REQUIRED.
func PreconditionRequired() *Error {
	return New(http.StatusPreconditionRequired, CodeIfMatchRequired, "If-Match header is required.")
}

// Unprocessable returns 422 with a business-rule code.
func Unprocessable(code, detail string) *Error {
	return New(http.StatusUnprocessableEntity, code, detail)
}

// TooManyRequests returns 429 with Retry-After.
func TooManyRequests(code string, retryAfterSeconds int) *Error {
	if code == "" {
		code = CodeRateLimited
	}
	e := New(http.StatusTooManyRequests, code, "Too many requests. Please retry later.")
	e.RetryAfterSeconds = retryAfterSeconds
	e.Retryable = true
	return e.With("retryAfterSeconds", retryAfterSeconds)
}

// Unavailable returns a retryable 503.
func Unavailable(code, detail string) *Error {
	if code == "" {
		code = CodeServiceUnavailable
	}
	e := New(http.StatusServiceUnavailable, code, detail)
	e.Retryable = true
	return e
}

// Internal returns 500 INTERNAL wrapping cause.
func Internal(cause error) *Error {
	return New(http.StatusInternalServerError, CodeInternal, "Something went wrong.").Wrap(cause)
}

// NotImplemented marks endpoints whose module is still a skeleton.
func NotImplemented(what string) *Error {
	return New(http.StatusNotImplemented, CodeNotImplemented, what+" is not implemented yet.")
}
