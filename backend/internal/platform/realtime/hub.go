package realtime

import "sync"

const (
	// subscriberBuffer bounds each connection's send queue; a slow consumer is dropped rather
	// than allowed to block the hub (08 §6.2).
	subscriberBuffer = 16
	maxFrameBytes    = 4096
)

// Subscription is a per-connection channel of frames for a set of topics.
type Subscription struct {
	C      chan Frame
	topics map[string]bool
	hub    *Hub
}

// Hub routes frames to local subscriptions by topic.
type Hub struct {
	mu      sync.RWMutex
	byTopic map[string]map[*Subscription]struct{}
	dropped uint64
}

// NewHub creates an empty hub.
func NewHub() *Hub { return &Hub{byTopic: map[string]map[*Subscription]struct{}{}} }

// Subscribe registers a connection for topics and returns its subscription.
func (h *Hub) Subscribe(topics []string) *Subscription {
	s := &Subscription{C: make(chan Frame, subscriberBuffer), topics: map[string]bool{}, hub: h}
	h.mu.Lock()
	for _, t := range topics {
		s.topics[t] = true
		if h.byTopic[t] == nil {
			h.byTopic[t] = map[*Subscription]struct{}{}
		}
		h.byTopic[t][s] = struct{}{}
	}
	h.mu.Unlock()
	return s
}

// Close removes the subscription from all topics.
func (s *Subscription) Close() {
	s.hub.mu.Lock()
	for t := range s.topics {
		if set := s.hub.byTopic[t]; set != nil {
			delete(set, s)
			if len(set) == 0 {
				delete(s.hub.byTopic, t)
			}
		}
	}
	s.hub.mu.Unlock()
}

// Dispatch delivers f to every local subscription on f.Topic. A full buffer drops the frame for
// that connection (it will refetch on reconnect).
func (h *Hub) Dispatch(f Frame) {
	h.capFrame(&f)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.byTopic[f.Topic] {
		select {
		case s.C <- f:
		default:
			h.dropped++
		}
	}
}

// LocalSubscribers returns how many local connections watch topic (presence counts).
func (h *Hub) LocalSubscribers(topic string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.byTopic[topic])
}

func (h *Hub) capFrame(f *Frame) {
	if len(f.Data) > maxFrameBytes {
		f.Data = []byte(`{}`)
	}
}
