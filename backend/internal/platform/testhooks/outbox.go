package testhooks

import (
	"sync"
	"time"
)

// Outbox captures messages the fake SMS/push/OTP providers would send, so E2E tests can read
// them instead of a real provider (20 §QA-16). It is process-local and used in non-prod only.
type Outbox struct {
	mu    sync.Mutex
	items []Message
}

// Message is one captured notification.
type Message struct {
	Channel   string            `json:"channel"` // SMS | PUSH | OTP
	To        string            `json:"to"`
	Code      string            `json:"code,omitempty"`
	Body      string            `json:"body,omitempty"`
	Meta      map[string]string `json:"meta,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

// Add records a message.
func (o *Outbox) Add(m Message) {
	o.mu.Lock()
	o.items = append(o.items, m)
	o.mu.Unlock()
}

// Latest returns the most recent message for channel+to, or false.
func (o *Outbox) Latest(channel, to string) (Message, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for i := len(o.items) - 1; i >= 0; i-- {
		if o.items[i].Channel == channel && o.items[i].To == to {
			return o.items[i], true
		}
	}
	return Message{}, false
}

// Drain returns and clears all captured messages.
func (o *Outbox) Drain() []Message {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := o.items
	o.items = nil
	return out
}
