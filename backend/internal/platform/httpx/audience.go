package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
)

type audienceKey struct{}

// Audiences maps a lower-cased Host to an audience (config.AppHosts).
type Audiences map[string]auth.Audience

// ResolveAudience derives the audience from Host (RV-001, RV-025). A client-sent X-Rovo-Audience
// is ignored. An unknown host yields 421 MISDIRECTED_REQUEST. The resolved audience is stored in
// the context for downstream middleware and handlers.
func (a Audiences) ResolveAudience() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := strings.ToLower(r.Host)
			aud, ok := a[host]
			if !ok {
				// Retry without the port for hosts configured bare.
				if h, _, found := strings.Cut(host, ":"); found {
					aud, ok = a[h]
				}
			}
			if !ok {
				Error(w, r, apperr.New(http.StatusMisdirectedRequest, apperr.CodeMisdirectedRequest,
					"Unknown host."))
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), audienceKey{}, aud)))
		})
	}
}

// AudienceFrom returns the audience resolved for the request.
func AudienceFrom(ctx context.Context) auth.Audience {
	a, _ := ctx.Value(audienceKey{}).(auth.Audience)
	return a
}
