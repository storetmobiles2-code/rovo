package httpx

import "net/http"

// Mux wraps http.ServeMux with method+pattern registration and a per-mux middleware stack. It
// keeps module route registration terse: m.Handle("POST /orders", h).
type Mux struct {
	mux *http.ServeMux
	mw  []Middleware
}

// NewMux creates a mux whose routes are wrapped by mw (applied per route so r.Pattern is set
// before the access log reads it).
func NewMux(mw ...Middleware) *Mux { return &Mux{mux: http.NewServeMux(), mw: mw} }

// Handle registers h for pattern ("METHOD /path").
func (m *Mux) Handle(pattern string, h http.HandlerFunc) {
	m.mux.Handle(pattern, Chain(h, m.mw...))
}

// Group returns a child mux sharing the underlying ServeMux but adding extra middleware.
func (m *Mux) Group(mw ...Middleware) *Mux {
	return &Mux{mux: m.mux, mw: append(append([]Middleware{}, m.mw...), mw...)}
}

// ServeHTTP implements http.Handler.
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) { m.mux.ServeHTTP(w, r) }

// Std returns the underlying ServeMux for mounting sub-handlers.
func (m *Mux) Std() *http.ServeMux { return m.mux }
