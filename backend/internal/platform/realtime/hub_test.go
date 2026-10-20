package realtime

import (
	"encoding/json"
	"testing"
)

func TestHubRoutesByTopic(t *testing.T) {
	h := NewHub()
	a := h.Subscribe([]string{"order:1", "user:1"})
	b := h.Subscribe([]string{"order:2"})
	if h.LocalSubscribers("order:1") != 1 {
		t.Fatal("presence count")
	}
	h.Dispatch(Frame{Topic: "order:1", Event: "order.status", Data: json.RawMessage(`{"v":1}`)})
	select {
	case f := <-a.C:
		if f.Event != "order.status" {
			t.Fatalf("event %s", f.Event)
		}
	default:
		t.Fatal("subscriber a should receive order:1")
	}
	select {
	case <-b.C:
		t.Fatal("subscriber b must not receive order:1")
	default:
	}
	// Unknown topic goes nowhere.
	h.Dispatch(Frame{Topic: "order:999"})
	a.Close()
	if h.LocalSubscribers("order:1") != 0 {
		t.Fatal("close must deregister")
	}
	// After close the topic map for user:1 is cleaned up too.
	if h.LocalSubscribers("user:1") != 0 {
		t.Fatal("close must deregister all topics")
	}
}

func TestHubDropsSlowConsumer(t *testing.T) {
	h := NewHub()
	s := h.Subscribe([]string{"t"})
	for i := 0; i < subscriberBuffer+10; i++ {
		h.Dispatch(Frame{Topic: "t"})
	}
	if len(s.C) != subscriberBuffer {
		t.Fatalf("buffer should be full at %d, got %d", subscriberBuffer, len(s.C))
	}
}

func TestCapFrame(t *testing.T) {
	h := NewHub()
	big := make([]byte, maxFrameBytes+1)
	f := Frame{Topic: "t", Data: big}
	h.capFrame(&f)
	if string(f.Data) != "{}" {
		t.Fatal("oversized data must be dropped")
	}
}
