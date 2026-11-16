package catalog

import (
	"fmt"
	"time"
)

// slot is one opening window on an ISO weekday (1 = Monday) in minutes since local midnight.
// closes <= opens means the window runs past midnight into the next day (doc 10).
type slot struct {
	day, opens, closes int
}

func (s slot) overnight() bool { return s.closes <= s.opens }

func isoWeekday(t time.Time) int {
	if d := int(t.Weekday()); d != 0 {
		return d
	}
	return 7
}

func hhmm(minute int) string { return fmt.Sprintf("%02d:%02d", minute/60, minute%60) }

// openAt reports whether a restaurant with these hours is open at local time, and if not, the
// local "HH:MM" of its next opening ("" when it has no hours at all).
func openAt(slots []slot, local time.Time) (open bool, opensAt string) {
	day, m := isoWeekday(local), local.Hour()*60+local.Minute()
	prev := (day+5)%7 + 1
	for _, s := range slots {
		switch {
		case s.day == day && !s.overnight() && m >= s.opens && m < s.closes,
			s.day == day && s.overnight() && m >= s.opens,
			s.day == prev && s.overnight() && m < s.closes:
			return true, ""
		}
	}
	// Next opening: later today first, then the following days in order.
	best := -1
	for offset := 0; offset <= 7 && best < 0; offset++ {
		d := (day-1+offset)%7 + 1
		for _, s := range slots {
			if s.day != d || (offset == 0 && s.opens <= m) {
				continue
			}
			if best < 0 || s.opens < best {
				best = s.opens
			}
		}
	}
	if best < 0 {
		return false, ""
	}
	return false, hhmm(best)
}
