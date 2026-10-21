//go:build testhooks

package testhooks

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"

	"github.com/storetmobiles2-code/rovo/backend/internal/platform/app"
	"github.com/storetmobiles2-code/rovo/backend/internal/platform/clock"
)

// Enabled reports whether the test-control surface is compiled in.
const Enabled = true

// Config holds the testhooks listener settings.
type Config struct {
	Addr   string
	Secret string
}

// Controls are the hooks the composition root hands the test surface: an advanceable clock, a
// function to run due River jobs, and the fake-provider outbox.
type Controls struct {
	Clock   *clock.Offset
	Outbox  *Outbox
	RunDue  func(ctx context.Context) (int, error)
	Reset   func(ctx context.Context) error
	Seed    func(ctx context.Context) error
}

// Mount registers /_test/* on mux. Every call requires the shared secret header.
func Mount(mux *http.ServeMux, cfg *Config, deps *app.Deps) {
	c, _ := deps.Service("testhooks").(*Controls)
	if c == nil {
		return
	}
	guard := func(h http.HandlerFunc) http.HandlerFunc {
		want := []byte(cfg.Secret)
		return func(w http.ResponseWriter, r *http.Request) {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Rovo-Testhooks-Secret")), want) != 1 {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			h(w, r)
		}
	}
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
	mux.HandleFunc("POST /_test/clock/advance", guard(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Advance string `json:"advance"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		d, err := time.ParseDuration(body.Advance)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c.Clock.Advance(d)
		writeJSON(w, map[string]any{"offsetSeconds": int(c.Clock.Offset().Seconds())})
	}))
	mux.HandleFunc("POST /_test/jobs/run-due", guard(func(w http.ResponseWriter, r *http.Request) {
		n, err := c.RunDue(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"ran": n})
	}))
	mux.HandleFunc("GET /_test/otp", guard(func(w http.ResponseWriter, r *http.Request) {
		m, ok := c.Outbox.Latest("OTP", r.URL.Query().Get("phone"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, m)
	}))
	mux.HandleFunc("GET /_test/outbox", guard(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, c.Outbox.Drain())
	}))
	mux.HandleFunc("POST /_test/reset", guard(func(w http.ResponseWriter, r *http.Request) {
		if c.Reset != nil {
			if err := c.Reset(r.Context()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if c.Seed != nil {
			_ = c.Seed(r.Context())
		}
		writeJSON(w, map[string]string{"status": "ok"})
	}))
}
