package clock

import (
	"testing"
	"time"
)

func TestFakeAndOffset(t *testing.T) {
	start := time.Date(2026, 10, 4, 13, 0, 0, 0, time.UTC)
	f := NewFake(start)
	if got := f.Advance(46 * time.Second); !got.Equal(start.Add(46 * time.Second)) {
		t.Fatalf("advance: %v", got)
	}
	o := NewOffset(f)
	o.Advance(time.Hour)
	if got := o.Now(); !got.Equal(start.Add(time.Hour + 46*time.Second)) {
		t.Fatalf("offset now: %v", got)
	}
	o.Set(0)
	if o.Offset() != 0 {
		t.Fatal("offset not reset")
	}
}

func TestPreviousISOWeek(t *testing.T) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Fatal(err)
	}
	// Tuesday 2026-10-06 10:00 IST → previous week Mon 2026-09-28 00:00 IST .. Mon 2026-10-05 00:00 IST.
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, ist)
	w := PreviousISOWeek(now, ist)
	wantStart := time.Date(2026, 9, 28, 0, 0, 0, 0, ist).UTC()
	wantEnd := time.Date(2026, 10, 5, 0, 0, 0, 0, ist).UTC()
	if !w.Start.Equal(wantStart) || !w.End.Equal(wantEnd) {
		t.Fatalf("got %v..%v want %v..%v", w.Start, w.End, wantStart, wantEnd)
	}
	// Monday 00:30 IST is just after the boundary: the previous week is the one that just ended.
	w2 := PreviousISOWeek(time.Date(2026, 10, 5, 0, 30, 0, 0, ist), ist)
	if !w2.End.Equal(wantEnd) {
		t.Fatalf("boundary: %v", w2.End)
	}
	if d := AccountingDate(time.Date(2026, 10, 4, 20, 0, 0, 0, time.UTC), ist); d.Day() != 5 {
		t.Fatalf("accounting date should roll to 5th in IST, got %v", d)
	}
}
