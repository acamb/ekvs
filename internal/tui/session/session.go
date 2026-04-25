// Package session holds the per-run authentication state for the TUI.
package session

import (
	"crypto"

	gossh "golang.org/x/crypto/ssh"
)

// Session stores the SSH identity loaded at runtime.
// The zero value is an unauthenticated session.
type Session struct {
	Signer      crypto.Signer
	PublicKey   gossh.PublicKey
	Fingerprint string
}

// IsAuthenticated reports whether the session holds a valid identity.
func (s *Session) IsAuthenticated() bool {
	return s.Signer != nil
}

// Clear resets the session, forgetting all loaded credentials.
func (s *Session) Clear() {
	s.Signer = nil
	s.PublicKey = nil
	s.Fingerprint = ""
}
