// Package money implements integer-paise arithmetic (00 §3). There are no floats anywhere in
// money paths; rounding is half-up (away from zero for negatives) at line level, and the payable
// rounds to the whole rupee half-up (R8, R58).
package money

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)

// Paise is an amount in paise (1/100 rupee).
type Paise int64

// CurrencyINR is the only currency in V1.
const CurrencyINR = "INR"

// ErrOverflow is returned when a computation does not fit in int64.
var ErrOverflow = errors.New("money: overflow")

// Money is the API representation {"amountPaise": 24900, "currency": "INR"} (11 API-D04).
type Money struct {
	AmountPaise int64  `json:"amountPaise"`
	Currency    string `json:"currency"`
}

// INR wraps p as Money.
func INR(p Paise) Money { return Money{AmountPaise: int64(p), Currency: CurrencyINR} }

// Paise returns the amount.
func (m Money) Paise() Paise { return Paise(m.AmountPaise) }

// Rupees builds an amount from whole rupees.
func Rupees(r int64) Paise { return Paise(r * 100) }

// Add returns p+q, failing on overflow.
func (p Paise) Add(q Paise) (Paise, error) {
	s := p + q
	if (q > 0 && s < p) || (q < 0 && s > p) {
		return 0, ErrOverflow
	}
	return s, nil
}

// MulDiv returns round_half_up(p * num / den). den must be positive.
func MulDiv(p Paise, num, den int64) (Paise, error) {
	if den <= 0 {
		return 0, errors.New("money: non-positive denominator")
	}
	n := new(big.Int).Mul(big.NewInt(int64(p)), big.NewInt(num))
	d := big.NewInt(den)
	q, r := new(big.Int).QuoRem(n, d, new(big.Int))
	// Half-up away from zero: |2r| >= d bumps the magnitude.
	twoR := new(big.Int).Abs(new(big.Int).Lsh(r, 1))
	if twoR.Cmp(d) >= 0 {
		if n.Sign() < 0 {
			q.Sub(q, big.NewInt(1))
		} else {
			q.Add(q, big.NewInt(1))
		}
	}
	if !q.IsInt64() {
		return 0, ErrOverflow
	}
	return Paise(q.Int64()), nil
}

// MulBps returns round_half_up(p * bps / 10000), e.g. 5% GST = 500 bps.
func (p Paise) MulBps(bps int64) Paise {
	v, err := MulDiv(p, bps, 10000)
	if err != nil {
		panic(err) // bps is bounded by CHECKs; overflow means corrupted input
	}
	return v
}

// InclusiveTax splits a tax-inclusive amount into (base, tax) for rateBps, base rounded half-up:
// base = round(gross × 10000 / (10000 + rate)), tax = gross − base (14 §10.3).
func InclusiveTax(gross Paise, rateBps int64) (base, tax Paise) {
	b, err := MulDiv(gross, 10000, 10000+rateBps)
	if err != nil {
		panic(err)
	}
	return b, gross - b
}

// RoundToRupee rounds p half-up to whole rupees and returns the rounded amount and the
// ROUND_OFF delta (rounded − p). For p ≥ 0 the delta lies in [−49, +50] paise.
func RoundToRupee(p Paise) (rounded, roundOff Paise) {
	r, err := MulDiv(p, 1, 100)
	if err != nil {
		panic(err)
	}
	rounded = r * 100
	return rounded, rounded - p
}

// SplitHalves splits an amount into two halves for CGST/SGST; the first half takes the odd paisa.
func SplitHalves(p Paise) (first, second Paise) {
	second = p / 2
	return p - second, second
}

// Format renders p as Indian rupees with lakh grouping: ₹1,23,456.78. Whole rupees omit the
// decimals when omitZeroPaise is true (₹250).
func Format(p Paise, omitZeroPaise bool) string {
	neg := p < 0
	v := uint64(p)
	if neg {
		if p == math.MinInt64 {
			v = uint64(math.MaxInt64) + 1
		} else {
			v = uint64(-p)
		}
	}
	rupees, paise := v/100, v%100
	s := groupIndian(fmt.Sprintf("%d", rupees))
	var b strings.Builder
	if neg {
		b.WriteString("-")
	}
	b.WriteString("₹")
	b.WriteString(s)
	if !(omitZeroPaise && paise == 0) {
		fmt.Fprintf(&b, ".%02d", paise)
	}
	return b.String()
}

func groupIndian(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	head, tail := digits[:len(digits)-3], digits[len(digits)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}
