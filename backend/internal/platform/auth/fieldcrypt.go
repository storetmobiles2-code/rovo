package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// ErrUnknownKey is returned when a ciphertext names a key id the keyring does not hold.
var ErrUnknownKey = errors.New("auth: unknown field-encryption key id")

// FieldKeyring encrypts PII columns (TOTP secrets, bank numbers, delivery codes) with
// AES-256-GCM. Each ciphertext is stored with the key id (`pii_key_id`) used, so keys rotate by
// adding a new active key and re-encrypting lazily (doc 10 §12).
type FieldKeyring struct {
	activeID string
	aeads    map[string]cipher.AEAD
}

// NewFieldKeyring builds a keyring; keys maps key id → 32-byte key.
func NewFieldKeyring(activeID string, keys map[string][]byte) (*FieldKeyring, error) {
	kr := &FieldKeyring{activeID: activeID, aeads: map[string]cipher.AEAD{}}
	for id, k := range keys {
		if len(k) != 32 {
			return nil, fmt.Errorf("auth: field key %q must be 32 bytes", id)
		}
		block, err := aes.NewCipher(k)
		if err != nil {
			return nil, err
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		kr.aeads[id] = aead
	}
	if _, ok := kr.aeads[activeID]; !ok {
		return nil, errors.New("auth: active field key missing")
	}
	return kr, nil
}

// ActiveKeyID returns the key id new ciphertexts use.
func (k *FieldKeyring) ActiveKeyID() string { return k.activeID }

// Encrypt seals plaintext; aad binds the ciphertext to its row/column (e.g. "admin_credentials.totp:<user id>")
// so a value copied to another row fails to decrypt. Output is nonce ‖ ciphertext.
func (k *FieldKeyring) Encrypt(plaintext, aad []byte) (ciphertext []byte, keyID string, err error) {
	aead := k.aeads[k.activeID]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, "", err
	}
	return aead.Seal(nonce, nonce, plaintext, aad), k.activeID, nil
}

// Decrypt opens a ciphertext produced by Encrypt with keyID.
func (k *FieldKeyring) Decrypt(ciphertext []byte, keyID string, aad []byte) ([]byte, error) {
	aead, ok := k.aeads[keyID]
	if !ok {
		return nil, ErrUnknownKey
	}
	if len(ciphertext) < aead.NonceSize()+aead.Overhead() {
		return nil, errors.New("auth: ciphertext too short")
	}
	nonce, ct := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]
	return aead.Open(nil, nonce, ct, aad)
}
