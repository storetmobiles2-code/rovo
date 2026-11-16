package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// RefreshTokenPrefix lets secret scanners spot leaked refresh tokens (12 §4.3).
const RefreshTokenPrefix = "rvr_"

// NewRefreshToken returns a new opaque refresh token and its SHA-256 hash for storage.
func NewRefreshToken() (token string, hash []byte) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: crypto/rand failed: " + err.Error())
	}
	token = RefreshTokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token)
}

// HashToken returns SHA-256(token). Full-entropy tokens need no slow hash.
func HashToken(token string) []byte {
	s := sha256.Sum256([]byte(token))
	return s[:]
}

// LooksLikeRefreshToken is a cheap shape check before a DB lookup.
func LooksLikeRefreshToken(token string) bool {
	return strings.HasPrefix(token, RefreshTokenPrefix) && len(token) == len(RefreshTokenPrefix)+43
}

// RandomToken returns a URL-safe random token of n bytes (setup links, invite codes).
func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("auth: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
