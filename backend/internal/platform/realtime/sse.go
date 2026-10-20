package realtime

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// SSE lifetime and heartbeat (R10, R52, R56).
const (
	Heartbeat   = 20 * time.Second
	MaxLifetime = 30 * time.Minute
)

// Writer streams Server-Sent Events to one client.
type Writer struct {
	w       http.ResponseWriter
	rc      *http.ResponseController
	onClose func()
}

// Start writes SSE headers and the initial retry hint. It fails if the response writer cannot
// flush.
func Start(w http.ResponseWriter) (*Writer, error) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	h.Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	sw := &Writer{w: w, rc: rc}
	fmt.Fprint(w, "retry: 3000\n\n")
	if err := rc.Flush(); err != nil {
		return nil, err
	}
	return sw, nil
}

// Event writes one SSE event frame.
func (s *Writer) Event(f Frame) error {
	if f.ID != "" {
		fmt.Fprintf(s.w, "id: %s\n", f.ID)
	}
	ev := f.Event
	if ev == "" {
		ev = "message"
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", ev, f.Data)
	return s.rc.Flush()
}

// Comment writes a heartbeat comment line.
func (s *Writer) Comment(text string) error {
	fmt.Fprintf(s.w, ": %s\n\n", text)
	return s.rc.Flush()
}

// control sends a named control event (ready, reauth, revoked, reconnect, degraded).
func (s *Writer) control(name, data string) error {
	if data == "" {
		data = "{}"
	}
	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, data)
	return s.rc.Flush()
}

// Ready announces the subscribed topics on (re)connect.
func (s *Writer) Ready(data string) error { return s.control("ready", data) }

// Reconnect asks the client to reconnect (30-minute rebalance).
func (s *Writer) Reconnect() error { return s.control("reconnect", "") }

// Revoked tells the client its session is gone.
func (s *Writer) Revoked() error { return s.control("revoked", "") }

// Pump streams frames from sub to the client with heartbeats until ctx is done, the lifetime cap
// is hit, or a write fails. sessionValid is polled on the heartbeat tick; returning false closes
// the stream with a revoked event (12 §4.7).
func (s *Writer) Pump(ctx context.Context, sub *Subscription, sessionValid func() bool) error {
	defer sub.Close()
	deadline := time.NewTimer(MaxLifetime)
	defer deadline.Stop()
	beat := time.NewTicker(Heartbeat)
	defer beat.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-deadline.C:
			return s.Reconnect()
		case <-beat.C:
			if sessionValid != nil && !sessionValid() {
				_ = s.Revoked()
				return nil
			}
			if err := s.Comment("ping"); err != nil {
				return err
			}
		case f, ok := <-sub.C:
			if !ok {
				return nil
			}
			if err := s.Event(f); err != nil {
				return err
			}
		}
	}
}
