package idgen

import (
	"regexp"
	"testing"
	"time"
)

var orderCodeRE = regexp.MustCompile(`^RV-[0-9A-HJKMNP-TV-Z]{6}$`)

func TestDefault(t *testing.T) {
	g := Default{}
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		id := g.NewID()
		if id.Version() != 7 {
			t.Fatalf("version %d", id.Version())
		}
		c := g.OrderCode()
		if !orderCodeRE.MatchString(c) {
			t.Fatalf("bad order code %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 1990 {
		t.Fatalf("too many collisions: %d unique", len(seen))
	}
	if tk := g.TicketCode(); !regexp.MustCompile(`^TK-[0-9A-HJKMNP-TV-Z]{6}$`).MatchString(tk) {
		t.Fatalf("bad ticket code %q", tk)
	}
}

func TestSequenceDeterministic(t *testing.T) {
	base := time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)
	a, b := NewSequence(base), NewSequence(base)
	for i := 0; i < 5; i++ {
		x, y := a.NewID(), b.NewID()
		if x != y || x.Version() != 7 {
			t.Fatalf("not deterministic or not v7: %s %s", x, y)
		}
	}
	if a.OrderCode() != "RV-000001" || a.OrderCode() != "RV-000002" {
		t.Fatal("sequence codes")
	}
	first, second := NewSequence(base).NewID(), NewSequence(base)
	second.NewID()
	if first.String() >= second.NewID().String() {
		t.Fatal("ids must sort by creation order")
	}
}
