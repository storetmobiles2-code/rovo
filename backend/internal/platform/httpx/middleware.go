package httpx

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/obs"
)

var defaultLogger = slog.Default()

// Middleware is a standard net/http decorator.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so the first listed runs outermost.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	return s.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach Flush for SSE.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// RequestContext assigns a request id (honouring an edge-provided X-Request-Id), binds a logger
// and sets the response header. First in the chain.
func RequestContext(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-Id")
			if id == "" || len(id) > 64 {
				id = uuid.NewString()
			}
			w.Header().Set("X-Request-Id", id)
			ctx := obs.WithRequestID(r.Context(), logger, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Recover turns a panic into 500 INTERNAL and logs the stack.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					obs.L(r.Context(), logger).Error("panic recovered", "panic", p, "path", r.URL.Path)
					Error(w, r, apperr.Internal(nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AccessLog logs one structured line per request and records metrics.
func AccessLog(logger *slog.Logger, m RequestObserver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			dur := time.Since(start)
			route := routePattern(r)
			obs.L(r.Context(), logger).Info("request",
				"method", r.Method, "route", route, "status", rec.status, "duration_ms", dur.Milliseconds())
			if m != nil {
				m.ObserveRequest(route, r.Method, rec.status, dur)
			}
		})
	}
}

// RequestObserver records request metrics (platform/obs.Metrics implements it).
type RequestObserver interface {
	ObserveRequest(route, method string, status int, d time.Duration)
}

func routePattern(r *http.Request) string {
	if r.Pattern != "" {
		if _, path, ok := strings.Cut(r.Pattern, " "); ok {
			return path
		}
		return r.Pattern
	}
	return "unmatched"
}

// SecurityHeaders sets conservative defaults for API responses (19).
func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			next.ServeHTTP(w, r)
		})
	}
}

// CrossOriginProtection applies Go 1.25 net/http.CrossOriginProtection (Fetch-Metadata / Origin)
// on unsafe cookie-authed methods (AUTH-D06). webhook/health handlers register their own chain
// without it.
func CrossOriginProtection() Middleware {
	p := http.NewCrossOriginProtection()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafe(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			if err := p.Check(r); err != nil {
				Error(w, r, apperr.Forbidden(apperr.CodeCSRFRejected, "Cross-origin request rejected."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireClientHeader requires X-Rovo-Client=<audience>-web on unsafe methods (AUTH-D06).
func RequireClientHeader() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafe(r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			aud := AudienceFrom(r.Context())
			got := r.Header.Get("X-Rovo-Client")
			if aud.IsApp() && got != aud.ClientHeaderValue() {
				Error(w, r, apperr.BadRequest(apperr.CodeClientHeaderRequired,
					"Missing or invalid X-Rovo-Client header."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimit caps request bodies.
func BodyLimit(max int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}

// OriginVerify requires the edge-injected X-Rovo-Origin-Verify secret on the public listener,
// stopping CDN/WAF bypass (RV-001). Disabled when secret == "" (local dev without an edge).
func OriginVerify(secret string) Middleware {
	want := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret != "" {
				got := []byte(r.Header.Get("X-Rovo-Origin-Verify"))
				if subtle.ConstantTimeCompare(got, want) != 1 {
					Error(w, r, apperr.Forbidden(apperr.CodeOriginNotVerified, "Origin verification failed."))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isSafe(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}

// Authenticator verifies access tokens (identity.Service / auth.Issuer implement a compatible
// method; the audience param comes from the Host).
type Authenticator interface {
	Verify(token string, aud auth.Audience) (*auth.Principal, error)
}

// SessionChecker reports whether a session is still valid (identity re-checks it, AUTH-D05).
type SessionChecker interface {
	SessionValid(sessionID uuid.UUID) bool
}
