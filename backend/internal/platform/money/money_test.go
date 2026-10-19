package money

import (
	"encoding/json"
	"testing"
)

func TestMulBpsHalfUp(t *testing.T) {
	cases := []struct {
		p    Paise
		bps  int64
		want Paise
	}{
		{33000, 250, 825}, // CGST 2.5% on ₹330 (11 §3 example)
		{10, 500, 1},      // 0.5 → 1 (half up)
		{9, 500, 0},       // 0.45 → 0
		{-10, 500, -1},    // symmetric away from zero
		{2500, 5000, 1250},
		{0, 1800, 0},
	}
	for _, c := range cases {
		if got := c.p.MulBps(c.bps); got != c.want {
			t.Errorf("%d×%dbps = %d, want %d", c.p, c.bps, got, c.want)
		}
	}
}

func TestInclusiveTax(t *testing.T) {
	// ₹30 delivery fee incl. 18% GST: base = round(3000×100/118) = 2542.37 → 2542, tax 458 (11 §3).
	base, tax := InclusiveTax(3000, 1800)
	if base != 2542 || tax != 458 {
		t.Fatalf("got %d/%d", base, tax)
	}
	a, b := SplitHalves(tax)
	if a+b != tax || a != 229 || b != 229 {
		t.Fatalf("halves %d %d", a, b)
	}
	a, b = SplitHalves(459)
	if a != 230 || b != 229 {
		t.Fatalf("odd halves %d %d", a, b)
	}
}

func TestRoundToRupee(t *testing.T) {
	cases := []struct{ in, out, off Paise }{
		{38150, 38200, 50}, // exactly half rounds up (R58)
		{38149, 38100, -49},
		{38100, 38100, 0},
		{38151, 38200, 49},
		{49, 0, -49},
		{50, 100, 50},
	}
	for _, c := range cases {
		r, off := RoundToRupee(c.in)
		if r != c.out || off != c.off {
			t.Errorf("RoundToRupee(%d) = %d,%d want %d,%d", c.in, r, off, c.out, c.off)
		}
	}
}

func TestMulDivOverflowAndErrors(t *testing.T) {
	if _, err := MulDiv(Paise(1<<62), 4, 1); err != ErrOverflow {
		t.Fatalf("want overflow, got %v", err)
	}
	if _, err := MulDiv(1, 1, 0); err == nil {
		t.Fatal("want error for zero denominator")
	}
	if _, err := Paise(1 << 62).Add(Paise(1 << 62)); err != ErrOverflow {
		t.Fatal("add overflow not detected")
	}
}

func TestFormat(t *testing.T) {
	cases := map[Paise]string{
		0:          "₹0.00",
		5:          "₹0.05",
		38100:      "₹381.00",
		12345678:   "₹1,23,456.78",
		1000000000: "₹1,00,00,000.00",
		-2550:      "-₹25.50",
	}
	for p, want := range cases {
		if got := Format(p, false); got != want {
			t.Errorf("Format(%d) = %q want %q", p, got, want)
		}
	}
	if got := Format(25000, true); got != "₹250" {
		t.Errorf("omit zero paise: %q", got)
	}
}

func TestMoneyJSON(t *testing.T) {
	b, err := json.Marshal(INR(38100))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"amountPaise":38100,"currency":"INR"}` {
		t.Fatalf("json %s", b)
	}
}
