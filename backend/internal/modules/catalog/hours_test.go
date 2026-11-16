package catalog

import (
	"testing"
	"time"
)

func TestOpenAt(t *testing.T) {
	// 2026-11-16 is a Monday.
	at := func(day, hour, minute int) time.Time {
		return time.Date(2026, 11, 15+day, hour, minute, 0, 0, time.UTC)
	}
	daily := func(opens, closes int) []slot {
		var out []slot
		for d := 1; d <= 7; d++ {
			out = append(out, slot{d, opens, closes})
		}
		return out
	}
	lunchDinner := []slot{{1, 12 * 60, 15 * 60}, {1, 19 * 60, 23 * 60}, {3, 12 * 60, 15 * 60}}
	lateNight := daily(18*60, 2*60) // 18:00–02:00

	cases := []struct {
		name    string
		slots   []slot
		when    time.Time
		open    bool
		opensAt string
	}{
		{"inside window", daily(11*60, 23*60), at(1, 12, 0), true, ""},
		{"opening minute is open", daily(11*60, 23*60), at(1, 11, 0), true, ""},
		{"closing minute is closed", daily(11*60, 23*60), at(1, 23, 0), false, "11:00"},
		{"before opening", daily(11*60, 23*60), at(1, 9, 30), false, "11:00"},
		{"between two slots", lunchDinner, at(1, 16, 0), false, "19:00"},
		{"after last slot, next open day is Wednesday", lunchDinner, at(1, 23, 30), false, "12:00"},
		{"closed day", lunchDinner, at(2, 13, 0), false, "12:00"},
		{"overnight before midnight", lateNight, at(1, 23, 59), true, ""},
		{"overnight after midnight", lateNight, at(2, 1, 30), true, ""},
		{"overnight closing minute", lateNight, at(2, 2, 0), false, "18:00"},
		{"overnight carries from Sunday into Monday", []slot{{7, 20 * 60, 60}}, at(1, 0, 30), true, ""},
		{"overnight slot does not open the same morning", []slot{{1, 20 * 60, 60}}, at(1, 0, 30), false, "20:00"},
		{"wraps to next week", []slot{{1, 9 * 60, 10 * 60}}, at(1, 12, 0), false, "09:00"},
		{"no hours", nil, at(1, 12, 0), false, ""},
	}
	for _, c := range cases {
		open, opensAt := openAt(c.slots, c.when)
		if open != c.open || opensAt != c.opensAt {
			t.Errorf("%s: openAt = (%v, %q), want (%v, %q)", c.name, open, opensAt, c.open, c.opensAt)
		}
	}
}
