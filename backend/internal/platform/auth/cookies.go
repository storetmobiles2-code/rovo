package auth

import (
	"net/http"
	"time"
)

// Cookie names. With Secure cookies (every deployed environment) the browser-enforced
// __Host-/__Secure- prefixes are used (12 §4.4); plain-HTTP local dev cannot set prefixed
// cookies, so it uses the bare names. Readers accept only the name matching the config.
const (
	accessCookieBase  = "rovo_at"
	refreshCookieBase = "rovo_rt"
	// RefreshCookiePath scopes the refresh cookie to the auth endpoints.
	RefreshCookiePath = "/api/v1/auth"
)

// Cookies writes and reads the session cookies for one deployment.
type Cookies struct {
	Secure bool
}

// AccessName returns the access cookie name.
func (c Cookies) AccessName() string {
	if c.Secure {
		return "__Host-" + accessCookieBase
	}
	return accessCookieBase
}

// RefreshName returns the refresh cookie name.
func (c Cookies) RefreshName() string {
	if c.Secure {
		return "__Secure-" + refreshCookieBase
	}
	return refreshCookieBase
}

func sameSite(aud Audience) http.SameSite {
	if aud == AudienceAdmin {
		return http.SameSiteStrictMode
	}
	return http.SameSiteLaxMode
}

// SetSession writes host-only access and refresh cookies (no Domain attribute).
func (c Cookies) SetSession(w http.ResponseWriter, aud Audience, access string, accessExp time.Time,
	refresh string, refreshMaxAge time.Duration, now time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: c.AccessName(), Value: access, Path: "/", HttpOnly: true, Secure: c.Secure,
		SameSite: sameSite(aud), MaxAge: maxAge(accessExp.Sub(now)),
	})
	if refresh != "" {
		http.SetCookie(w, &http.Cookie{
			Name: c.RefreshName(), Value: refresh, Path: RefreshCookiePath, HttpOnly: true, Secure: c.Secure,
			SameSite: http.SameSiteStrictMode, MaxAge: maxAge(refreshMaxAge),
		})
	}
}

// Clear expires both cookies (logout, revoked session).
func (c Cookies) Clear(w http.ResponseWriter, aud Audience) {
	http.SetCookie(w, &http.Cookie{Name: c.AccessName(), Value: "", Path: "/", HttpOnly: true, Secure: c.Secure,
		SameSite: sameSite(aud), MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: c.RefreshName(), Value: "", Path: RefreshCookiePath, HttpOnly: true,
		Secure: c.Secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

// AccessToken returns the access token cookie value.
func (c Cookies) AccessToken(r *http.Request) string {
	if ck, err := r.Cookie(c.AccessName()); err == nil {
		return ck.Value
	}
	return ""
}

// RefreshToken returns the refresh token cookie value.
func (c Cookies) RefreshToken(r *http.Request) string {
	if ck, err := r.Cookie(c.RefreshName()); err == nil {
		return ck.Value
	}
	return ""
}

func maxAge(d time.Duration) int {
	s := int(d / time.Second)
	if s < 1 {
		return 1
	}
	return s
}
