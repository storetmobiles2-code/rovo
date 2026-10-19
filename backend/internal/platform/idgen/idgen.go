// Package idgen generates UUIDv7 primary keys (doc 10 DB-D02) and human-facing codes such as
// order codes RV-XXXXXX and ticket codes TK-XXXXXX in Crockford base32.
package idgen

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/google/uuid"
)

// CrockfordAlphabet excludes I, L, O and U, matching the DB CHECK '^RV-[0-9A-HJKMNP-TV-Z]{6}$'.
const CrockfordAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// Generator creates identifiers. Inject it; never call uuid.New in modules.
type Generator interface {
	NewID() uuid.UUID
	OrderCode() string
	TicketCode() string
}

// Default produces random UUIDv7 values and random codes. Codes are not guaranteed unique:
// callers insert and retry on a unique violation (32^6 ≈ 1.07e9 values).
type Default struct{}

// NewID returns a UUIDv7.
func (Default) NewID() uuid.UUID {
	for {
		id, err := uuid.NewV7()
		if err == nil {
			return id
		}
	}
}

// OrderCode returns "RV-" + 6 Crockford characters.
func (Default) OrderCode() string { return "RV-" + RandomCrockford(6) }

// TicketCode returns "TK-" + 6 Crockford characters.
func (Default) TicketCode() string { return "TK-" + RandomCrockford(6) }

// RandomCrockford returns n characters from the Crockford alphabet using crypto/rand.
func RandomCrockford(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("idgen: crypto/rand failed: " + err.Error())
	}
	out := make([]byte, n)
	for i, b := range buf {
		out[i] = CrockfordAlphabet[b&31] // 256 is a multiple of 32: no modulo bias
	}
	return string(out)
}

// Sequence is a deterministic generator for tests: IDs are valid UUIDv7 values with an
// increasing timestamp, codes count up from RV-000001.
type Sequence struct {
	mu   sync.Mutex
	base time.Time
	n    uint64
	code uint64
}

// NewSequence starts a deterministic generator at base.
func NewSequence(base time.Time) *Sequence { return &Sequence{base: base} }

// NewID returns the next deterministic UUIDv7.
func (s *Sequence) NewID() uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	var u uuid.UUID
	ms := uint64(s.base.UnixMilli()) + s.n
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)
	binary.BigEndian.PutUint64(u[8:], s.n)
	u[6] = 0x70 | (u[6] & 0x0f) // version 7
	u[8] = 0x80 | (u[8] & 0x3f) // RFC 4122 variant
	return u
}

// OrderCode returns RV-000001, RV-000002, ...
func (s *Sequence) OrderCode() string { return "RV-" + s.nextCode() }

// TicketCode returns TK-000001, ...
func (s *Sequence) TicketCode() string { return "TK-" + s.nextCode() }

func (s *Sequence) nextCode() string {
	s.mu.Lock()
	s.code++
	v := s.code
	s.mu.Unlock()
	out := make([]byte, 6)
	for i := 5; i >= 0; i-- {
		out[i] = CrockfordAlphabet[v%32]
		v /= 32
	}
	return string(out)
}
