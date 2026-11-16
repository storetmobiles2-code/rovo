package httpx

import (
	"net/http"
	"strconv"
	"time"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/ratelimit"
)

// RateLimit applies a Postgres-backed budget keyed by keyFn. It sets the IETF draft RateLimit
// headers and returns 429 with Retry-After when exceeded (11 §1.10). keyFn returns "" to skip.
func RateLimit(lim *ratelimit.Limiter, policy string, limit int, window time.Duration, keyFn func(*http.Request) string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}
			d, err := lim.Allow(r.Context(), key, limit, window)
			if err != nil {
				// Fail open on limiter errors so a DB hiccup does not take down the API; the edge
				// WAF is the hard limit. The error is logged by the caller's access log.
				next.ServeHTTP(w, r)
				return
			}
			h := w.Header()
			h.Set("RateLimit-Policy", "\""+policy+"\";q="+strconv.Itoa(limit)+";w="+strconv.Itoa(int(window/time.Second)))
			h.Set("RateLimit", "\""+policy+"\";r="+strconv.Itoa(d.Remaining)+";t="+strconv.Itoa(int(time.Until(d.ResetAt).Seconds())))
			if !d.Allowed {
				Error(w, r, apperr.TooManyRequests(apperr.CodeRateLimited, int(d.RetryAfter.Seconds())+1))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
