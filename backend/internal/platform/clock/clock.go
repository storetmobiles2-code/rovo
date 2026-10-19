// Package clock provides the injectable application clock. Business code never calls
// time.Now() directly (R19): deadlines, cut-offs and River ScheduledAt values come from a Clock
// so tests and the testhooks build can move time.
package clock

import (
	"sync"
	"sync/atomic"
	"time"
)

// Clock returns the current instant in UTC.
type Clock interface {
	Now() time.Time
}

// Real is the wall clock.
type Real struct{}

// Now returns time.Now in UTC.
func (Real) Now() time.Time { return time.Now().UTC() }

// Offset is the wall clock shifted by an adjustable offset. Production runs with a zero offset;
// testhooks builds advance it so api and worker processes agree on a "fake now".
type Offset struct {
	base   Clock
	offset atomic.Int64
}

// NewOffset wraps base (Real when nil).
func NewOffset(base Clock) *Offset {
	if base == nil {
		base = Real{}
	}
	return &Offset{base: base}
}

// Now returns base time plus the offset.
func (o *Offset) Now() time.Time { return o.base.Now().Add(time.Duration(o.offset.Load())) }

// Set replaces the offset.
func (o *Offset) Set(d time.Duration) { o.offset.Store(int64(d)) }

// Advance adds d to the offset and returns the new offset.
func (o *Offset) Advance(d time.Duration) time.Duration {
	return time.Duration(o.offset.Add(int64(d)))
}

// Offset returns the current offset.
func (o *Offset) Offset() time.Duration { return time.Duration(o.offset.Load()) }

// Fake is a manually driven clock for unit tests.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

// NewFake starts a fake clock at t.
func NewFake(t time.Time) *Fake { return &Fake{now: t.UTC()} }

// Now returns the fake time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Set moves the clock to t.
func (f *Fake) Set(t time.Time) {
	f.mu.Lock()
	f.now = t.UTC()
	f.mu.Unlock()
}

// Advance moves the clock forward by d and returns the new time.
func (f *Fake) Advance(d time.Duration) time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.now = f.now.Add(d)
	return f.now
}
