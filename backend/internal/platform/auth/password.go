package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params are the argon2id cost parameters (12 §3.1: m=64 MiB, t=3, p=1).
type Argon2Params struct {
	MemoryKiB uint32
	Time      uint32
	Threads   uint8
	SaltLen   int
	KeyLen    uint32
}

// DefaultArgon2 is the production cost.
var DefaultArgon2 = Argon2Params{MemoryKiB: 64 * 1024, Time: 3, Threads: 1, SaltLen: 16, KeyLen: 32}

// ErrPasswordMismatch is returned for a wrong password.
var ErrPasswordMismatch = errors.New("auth: password mismatch")

// PasswordHasher hashes with argon2id; a semaphore caps concurrent hashes per replica so a burst
// of logins cannot exhaust memory (4 × 64 MiB).
type PasswordHasher struct {
	params Argon2Params
	sem    chan struct{}
}

// NewPasswordHasher creates a hasher allowing maxConcurrent hashes at once.
func NewPasswordHasher(p Argon2Params, maxConcurrent int) *PasswordHasher {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &PasswordHasher{params: p, sem: make(chan struct{}, maxConcurrent)}
}

func (h *PasswordHasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *PasswordHasher) release() { <-h.sem }

// Hash returns a PHC string: $argon2id$v=19$m=65536,t=3,p=1$<salt>$<hash>.
func (h *PasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()
	salt := make([]byte, h.params.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, h.params.Time, h.params.MemoryKiB, h.params.Threads, h.params.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, h.params.MemoryKiB, h.params.Time,
		h.params.Threads, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

// Verify checks password against a PHC hash. needsRehash is true when the stored parameters
// differ from the hasher's (re-hash on successful login, 12 §3.1).
func (h *PasswordHasher) Verify(ctx context.Context, phc, password string) (needsRehash bool, err error) {
	p, salt, want, err := parsePHC(phc)
	if err != nil {
		return false, err
	}
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer h.release()
	got := argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKiB, p.Threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, ErrPasswordMismatch
	}
	return p.MemoryKiB != h.params.MemoryKiB || p.Time != h.params.Time || p.Threads != h.params.Threads, nil
}

func parsePHC(phc string) (Argon2Params, []byte, []byte, error) {
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, errors.New("auth: unsupported password hash")
	}
	var v int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return Argon2Params{}, nil, nil, errors.New("auth: unsupported argon2 version")
	}
	var p Argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Time, &p.Threads); err != nil {
		return Argon2Params{}, nil, nil, errors.New("auth: malformed argon2 parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Argon2Params{}, nil, nil, errors.New("auth: malformed salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) < 16 {
		return Argon2Params{}, nil, nil, errors.New("auth: malformed hash")
	}
	return p, salt, key, nil
}
