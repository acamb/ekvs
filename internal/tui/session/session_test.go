package session

import (
	"crypto"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	internalssh "ekvs/internal/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// testdataDir returns the absolute path to internal/ssh/testdata.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "internal", "ssh", "testdata")
	abs, _ := filepath.Abs(root)
	return abs
}

// loadSession parses the PEM key at testdataDir/name and calls SetAuthenticated.
func loadSession(t *testing.T, name string) Session {
	t.Helper()
	pem, err := os.ReadFile(filepath.Join(testdataDir(), name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pem)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	var s Session
	if err := s.SetAuthenticated(signer, pub, internalssh.Fingerprint(pub)); err != nil {
		t.Fatalf("SetAuthenticated(%s): %v", name, err)
	}
	return s
}

// ── Zero value ────────────────────────────────────────────────────────────────

func TestSession_ZeroValueIsUnauthenticated(t *testing.T) {
	var s Session
	if s.IsAuthenticated() {
		t.Error("zero-value session should not be authenticated")
	}
}

// ── SetAuthenticated ──────────────────────────────────────────────────────────

func TestSession_SetAuthenticated_SupportedKeys(t *testing.T) {
	for _, name := range []string{"ed25519", "ecdsa", "rsa"} {
		t.Run(name, func(t *testing.T) {
			s := loadSession(t, name)
			if !s.IsAuthenticated() {
				t.Error("expected authenticated")
			}
			if s.Fingerprint == "" {
				t.Error("expected non-empty fingerprint")
			}
		})
	}
}

func TestSession_SetAuthenticated_UnsupportedKey(t *testing.T) {
	pem, err := os.ReadFile(filepath.Join(testdataDir(), "ed25519"))
	if err != nil {
		t.Fatalf("read ed25519: %v", err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pem)
	if err != nil {
		t.Fatalf("parse ed25519: %v", err)
	}
	// Wrap the signer so DeriveKey's type switch hits the default case.
	wrapped := &opaqueSignerWrapper{inner: signer}
	var s Session
	err = s.SetAuthenticated(wrapped, pub, "")
	if err == nil {
		t.Fatal("expected error for unsupported key type, got nil")
	}
	if s.IsAuthenticated() {
		t.Error("session must not be authenticated after failed SetAuthenticated")
	}
}

// opaqueSignerWrapper wraps a crypto.Signer but is itself an unknown type,
// so encryption.DeriveKey's type switch falls through to the default case.
type opaqueSignerWrapper struct{ inner crypto.Signer }

func (o *opaqueSignerWrapper) Public() crypto.PublicKey { return o.inner.Public() }
func (o *opaqueSignerWrapper) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return o.inner.Sign(rand, digest, opts)
}

// ── Encrypt / Decrypt round-trips ─────────────────────────────────────────────

func TestSession_EncryptDecrypt_RoundTrip(t *testing.T) {
	const plaintext = "super-secret-value-42"

	for _, name := range []string{"ed25519", "ecdsa", "rsa"} {
		t.Run(name, func(t *testing.T) {
			s := loadSession(t, name)

			ct, err := s.Encrypt(plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			if ct == plaintext {
				t.Error("ciphertext must differ from plaintext")
			}

			got, err := s.Decrypt(ct)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if got != plaintext {
				t.Errorf("round-trip mismatch: got %q, want %q", got, plaintext)
			}
		})
	}
}

func TestSession_EncryptDecrypt_DeterministicKey(t *testing.T) {
	// Two sessions loaded from the same key must produce cross-decryptable output.
	s1 := loadSession(t, "ed25519")
	s2 := loadSession(t, "ed25519")

	ct, _ := s1.Encrypt("hello")
	got, err := s2.Decrypt(ct)
	if err != nil {
		t.Fatalf("cross-session Decrypt: %v", err)
	}
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
}

// ── Error paths ───────────────────────────────────────────────────────────────

func TestSession_Encrypt_UnauthenticatedReturnsError(t *testing.T) {
	var s Session
	_, err := s.Encrypt("x")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestSession_Decrypt_UnauthenticatedReturnsError(t *testing.T) {
	var s Session
	_, err := s.Decrypt("dGVzdA==")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestSession_Decrypt_TamperedCiphertext(t *testing.T) {
	s := loadSession(t, "ed25519")
	ct, _ := s.Encrypt("data")
	// Flip the last byte of the base64 blob.
	tampered := ct[:len(ct)-1] + "X"
	_, err := s.Decrypt(tampered)
	if !errors.Is(err, ErrDecrypt) {
		t.Errorf("expected ErrDecrypt, got %v", err)
	}
}

// ── Clear ─────────────────────────────────────────────────────────────────────

func TestSession_Clear(t *testing.T) {
	s := loadSession(t, "ed25519")
	if !s.IsAuthenticated() {
		t.Fatal("expected authenticated before clear")
	}
	s.Clear()
	if s.IsAuthenticated() {
		t.Error("expected unauthenticated after clear")
	}
	if s.Fingerprint != "" {
		t.Error("fingerprint should be empty after clear")
	}
	_, err := s.Encrypt("x")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("Encrypt after Clear: expected ErrNotAuthenticated, got %v", err)
	}
	_, err = s.Decrypt("dGVzdA==")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("Decrypt after Clear: expected ErrNotAuthenticated, got %v", err)
	}
}

// ── Type check: gossh.PublicKey in session ────────────────────────────────────

func TestSession_PublicKeySet(t *testing.T) {
	s := loadSession(t, "ed25519")
	if s.PublicKey == nil {
		t.Error("expected non-nil PublicKey after SetAuthenticated")
	}
	if _, ok := s.PublicKey.(gossh.PublicKey); !ok {
		t.Error("PublicKey must implement gossh.PublicKey")
	}
}
