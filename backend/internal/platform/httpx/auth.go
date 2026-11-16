package httpx

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/apperr"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/auth"
)

// AuthConfig wires the principal-extraction middleware.
type AuthConfig struct {
	Issuer   Authenticator
	Cookies  auth.Cookies
	Sessions SessionChecker // optional; checked for non-customer audiences
}

// Authenticate extracts the principal from the host-only access cookie (app hosts) or the bearer
// header (api host / native, later). It never mixes the two: cookies are ignored on the webhook
// audience and Authorization is ignored on app hosts (12 §4.5). Anonymous requests continue with
// no principal; endpoints enforce auth with Require*.
func Authenticate(cfg AuthConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			aud := AudienceFrom(r.Context())
			var token string
			switch {
			case aud.IsApp():
				token = cfg.Cookies.AccessToken(r)
			case aud == auth.AudienceWebhook:
				// Webhooks authenticate by signature; no principal.
				next.ServeHTTP(w, r)
				return
			}
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}
			p, err := cfg.Issuer.Verify(token, aud)
			if err != nil {
				Error(w, r, mapTokenError(err))
				return
			}
			if cfg.Sessions != nil && aud != auth.AudienceCustomer {
				if !cfg.Sessions.SessionValid(p.SessionID) {
					cfg.Cookies.Clear(w, aud)
					Error(w, r, apperr.Unauthenticated(apperr.CodeSessionRevoked, "Your session has ended."))
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), p)))
		})
	}
}

func mapTokenError(err error) error {
	switch err {
	case auth.ErrTokenExpired:
		return apperr.Unauthenticated(apperr.CodeTokenExpired, "Your session expired.")
	case auth.ErrTokenAudienceMismatch:
		return apperr.Unauthenticated(apperr.CodeTokenAudienceMismatch, "Token not valid for this app.")
	default:
		return apperr.Unauthenticated(apperr.CodeUnauthenticated, "Authentication required.")
	}
}

// Principal returns the request principal or nil.
func Principal(r *http.Request) *auth.Principal { return auth.PrincipalFrom(r.Context()) }

// RequireAuth wraps h so anonymous requests get 401.
func RequireAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if Principal(r) == nil {
			Error(w, r, apperr.Unauthenticated(apperr.CodeUnauthenticated, "Authentication required."))
			return
		}
		h(w, r)
	}
}

// Require wraps h so only principals holding one of roles may proceed (RBAC L1). Anonymous → 401,
// wrong role → 403 (11 §1.8: 403 only when the caller is known but not allowed).
func Require(roles ...auth.Role) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			p := Principal(r)
			if p == nil {
				Error(w, r, apperr.Unauthenticated(apperr.CodeUnauthenticated, "Authentication required."))
				return
			}
			if !p.HasRole(roles...) {
				Error(w, r, apperr.Forbidden(apperr.CodeForbidden, "You do not have access to this action."))
				return
			}
			h(w, r)
		}
	}
}

// RequireAdmin allows any admin sub-role.
func RequireAdmin(h http.HandlerFunc) http.HandlerFunc { return Require(auth.AdminRoles...)(h) }

// BearerToken returns the bearer token on the api host (native clients, later).
func BearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(h, "Bearer "); ok {
		return after
	}
	return ""
}

var _ = uuid.Nil
