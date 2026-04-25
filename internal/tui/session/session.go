// Package session holds the per-run authentication state for the TUI.
package session

import (
	"crypto"
	"errors"
	"fmt"

	"ekvs/internal/encryption"

	gossh "golang.org/x/crypto/ssh"
)

var (
	// ErrNotAuthenticated is returned when Encrypt or Decrypt is called on an
	// unauthenticated session.
	ErrNotAuthenticated = errors.New("session: not authenticated")

	// ErrEncrypt wraps encryption errors from internal/encryption.
	ErrEncrypt = errors.New("session: encrypt failed")

	// ErrDecrypt wraps decryption errors from internal/encryption.
	ErrDecrypt = errors.New("session: decrypt failed")
)

// Session stores the SSH identity loaded at runtime.
// The zero value is an unauthenticated session.
type Session struct {
	Signer      crypto.Signer
	PublicKey   gossh.PublicKey
	Fingerprint string
	encKey      []byte // 32-byte AES-256 key; nil until authenticated
}

// SetAuthenticated authenticates the session and derives the symmetric
// encryption key from signer. Returns an error if key derivation fails.
func (s *Session) SetAuthenticated(signer crypto.Signer, pub gossh.PublicKey, fp string) error {
	key, err := encryption.DeriveKey(signer)
	if err != nil {
		return fmt.Errorf("session: derive key: %w", err)
	}
	s.Signer = signer
	s.PublicKey = pub
	s.Fingerprint = fp
	s.encKey = key
	return nil
}

// IsAuthenticated reports whether the session holds a valid identity.
func (s *Session) IsAuthenticated() bool {
	return s.Signer != nil
}

// Encrypt encrypts plaintext using the session's derived AES-256-GCM key.
// Returns ErrNotAuthenticated if the session is not authenticated.
func (s *Session) Encrypt(plaintext string) (string, error) {
	if s.encKey == nil {
		return "", ErrNotAuthenticated
	}
	ct, err := encryption.Encrypt(s.encKey, []byte(plaintext))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrEncrypt, err)
	}
	return ct, nil
}

// Decrypt decrypts a base64-encoded ciphertext blob produced by Encrypt.
// Returns ErrNotAuthenticated if the session is not authenticated.
func (s *Session) Decrypt(encoded string) (string, error) {
	if s.encKey == nil {
		return "", ErrNotAuthenticated
	}
	pt, err := encryption.Decrypt(s.encKey, encoded)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecrypt, err)
	}
	return string(pt), nil
}

// Clear resets the session, zeroing the encryption key before clearing all fields.
func (s *Session) Clear() {
	for i := range s.encKey {
		s.encKey[i] = 0
	}
	*s = Session{}
}
