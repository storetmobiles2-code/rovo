package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// Keyset holds the active Ed25519 signing key and previous verify-only keys (12 §4.10).
type Keyset struct {
	activeKID string
	active    ed25519.PrivateKey
	verify    map[string]ed25519.PublicKey
}

// KID derives a key id: base64url of the first 8 bytes of SHA-256(public key).
func KID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return base64.RawURLEncoding.EncodeToString(sum[:8])
}

// NewKeyset builds a keyset from the active private key and optional previous public keys.
func NewKeyset(active ed25519.PrivateKey, previous ...ed25519.PublicKey) (*Keyset, error) {
	if len(active) != ed25519.PrivateKeySize {
		return nil, errors.New("auth: invalid ed25519 private key")
	}
	pub := active.Public().(ed25519.PublicKey)
	ks := &Keyset{activeKID: KID(pub), active: active, verify: map[string]ed25519.PublicKey{}}
	ks.verify[ks.activeKID] = pub
	for _, p := range previous {
		if len(p) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("auth: invalid previous public key")
		}
		ks.verify[KID(p)] = p
	}
	return ks, nil
}

// ActiveKID returns the signing key id.
func (k *Keyset) ActiveKID() string { return k.activeKID }

// PublicKeys returns kid → public key for JWKS publication on the internal listener.
func (k *Keyset) PublicKeys() map[string]ed25519.PublicKey {
	out := make(map[string]ed25519.PublicKey, len(k.verify))
	for kid, p := range k.verify {
		out[kid] = p
	}
	return out
}
