package clock

import "time"

// Week is a half-open local-time interval [Start, End) expressed as UTC instants.
type Week struct {
	Start time.Time
	End   time.Time
}

// StartOfDay returns local midnight of t's calendar day in loc, as UTC.
func StartOfDay(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc).UTC()
}

// StartOfISOWeek returns Monday 00:00 (in loc) of the week containing t, as UTC.
func StartOfISOWeek(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	offset := (int(l.Weekday()) + 6) % 7 // Monday = 0
	d := time.Date(l.Year(), l.Month(), l.Day()-offset, 0, 0, 0, 0, loc)
	return d.UTC()
}

// PreviousISOWeek returns the last complete Mon–Sun week before t in loc. Settlement catch-up
// jobs (13 §5.2) use it to decide which period must already have a batch.
func PreviousISOWeek(t time.Time, loc *time.Location) Week {
	end := StartOfISOWeek(t, loc)
	start := end.In(loc).AddDate(0, 0, -7)
	return Week{Start: time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc).UTC(), End: end}
}

// AccountingDate returns the calendar date of t in loc (ledger_journals.accounting_date).
func AccountingDate(t time.Time, loc *time.Location) time.Time {
	l := t.In(loc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
}
