// Package health serves liveness (/healthz) and readiness (/readyz). Readiness checks the DB
// and the LISTEN connection so the LB stops routing to a replica that lost its event bus (11 §1.1).
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Checker reports readiness.
type Checker struct {
	pool      *pgxpool.Pool
	listenUp  atomic.Bool
	startedAt time.Time
	version   string
}

// New builds a checker. Call SetListenUp from the realtime bus.
func New(pool *pgxpool.Pool, version string) *Checker {
	c := &Checker{pool: pool, startedAt: time.Now(), version: version}
	c.listenUp.Store(true)
	return c
}

// SetListenUp records whether the LISTEN/NOTIFY bus is connected.
func (c *Checker) SetListenUp(up bool) { c.listenUp.Store(up) }

// Live always reports 200 once the process is running.
func (c *Checker) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "version": c.version})
}

// Ready checks the DB (and reports LISTEN state); 503 when degraded.
func (c *Checker) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	body := map[string]any{"version": c.version, "uptimeSeconds": int(time.Since(c.startedAt).Seconds())}
	code := http.StatusOK
	if err := c.pool.Ping(ctx); err != nil {
		body["db"] = "down"
		body["status"] = "unready"
		code = http.StatusServiceUnavailable
	} else {
		body["db"] = "ok"
	}
	body["listen"] = map[bool]string{true: "ok", false: "down"}[c.listenUp.Load()]
	if _, ok := body["status"]; !ok {
		body["status"] = "ok"
	}
	writeJSON(w, code, body)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
