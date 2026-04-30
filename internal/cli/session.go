package cli

import (
	"crypto"
	"errors"
	"fmt"

	"ekvs/internal/encryption"
)

// Sentinel errors for session operations.
var (
	// ErrNotAuthenticated is returned when Decrypt is called on a Session
	// whose encryption key has not been initialised.
	ErrNotAuthenticated = errors.New("not authenticated: no encryption key")

	// ErrDecrypt wraps any decryption failure produced by the encryption layer.
	ErrDecrypt = errors.New("decrypt failed")
)

// Session holds the derived encryption key for a CLI session.
type Session struct {
	encKey []byte
}

// NewSession derives an AES-256 key from the provided signer and returns a
// ready-to-use Session. Returns an error if key derivation fails (e.g. the
// signer's key type is not supported).
func NewSession(signer crypto.Signer) (*Session, error) {
	key, err := encryption.DeriveKey(signer)
	if err != nil {
		return nil, fmt.Errorf("deriving session key: %w", err)
	}
	return &Session{encKey: key}, nil
}

// Decrypt decrypts a base64-encoded ciphertext that was produced by
// encryption.Encrypt using the session key.
//
// Returns ErrNotAuthenticated if the session has no key, or a wrapped
// ErrDecrypt on any decryption failure.
func (s *Session) Decrypt(encoded string) (string, error) {
	if s.encKey == nil {
		return "", ErrNotAuthenticated
	}
	plaintext, err := encryption.Decrypt(s.encKey, encoded)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDecrypt, err)
	}
	return string(plaintext), nil
}
