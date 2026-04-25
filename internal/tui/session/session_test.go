package session

import (
	"crypto"
	"io"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

// fakeSigner is a minimal crypto.Signer for testing.
type fakeSigner struct{ crypto.PublicKey }

func (f fakeSigner) Public() crypto.PublicKey { return f.PublicKey }
func (f fakeSigner) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	return nil, nil
}

func TestSession_ZeroValueIsUnauthenticated(t *testing.T) {
	var s Session
	if s.IsAuthenticated() {
		t.Error("zero-value session should not be authenticated")
	}
}

func TestSession_Clear(t *testing.T) {
	s := Session{
		Signer:      fakeSigner{},
		PublicKey:   gossh.PublicKey(nil),
		Fingerprint: "SHA256:abc",
	}
	if !s.IsAuthenticated() {
		t.Error("session with signer should be authenticated")
	}
	s.Clear()
	if s.IsAuthenticated() {
		t.Error("cleared session should not be authenticated")
	}
	if s.Fingerprint != "" {
		t.Error("fingerprint should be empty after clear")
	}
}
