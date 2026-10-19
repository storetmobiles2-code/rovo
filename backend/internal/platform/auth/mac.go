package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// MAC returns HMAC-SHA-256(key, len-prefixed parts). Length prefixes prevent ambiguity between
// ("ab","c") and ("a","bc"). Used for OTP codes (challenge id ‖ code), phone blind keys and
// signed cursors/quote ids.
func MAC(key []byte, parts ...[]byte) []byte {
	m := hmac.New(sha256.New, key)
	var l [4]byte
	for _, p := range parts {
		binary.BigEndian.PutUint32(l[:], uint32(len(p)))
		m.Write(l[:])
		m.Write(p)
	}
	return m.Sum(nil)
}

// EqualMAC compares MACs in constant time.
func EqualMAC(a, b []byte) bool { return hmac.Equal(a, b) }
