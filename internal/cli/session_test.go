package cli

import (
	"crypto"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"ekvs/internal/encryption"
	internalssh "ekvs/internal/ssh"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func testdataPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	return filepath.Join(dir, "..", "ssh", "testdata", name)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(testdataPath(name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	return b
}

func signerFromFixture(t *testing.T, name string) crypto.Signer {
	t.Helper()
	signer, _, err := internalssh.ParsePrivateKey(readFixture(t, name))
	if err != nil {
		t.Fatalf("ParsePrivateKey(%s): %v", name, err)
	}
	return signer
}

// unsupportedSigner satisfies crypto.Signer with an unknown public key type.
type unsupportedSigner struct{}

func (unsupportedSigner) Public() crypto.PublicKey { return struct{}{} }
func (unsupportedSigner) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	return nil, nil
}

// ── NewSession ────────────────────────────────────────────────────────────────

func TestNewSession_SupportedKeys(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{"ed25519", "ed25519"},
		{"ecdsa", "ecdsa"},
		{"rsa", "rsa"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			signer := signerFromFixture(t, tc.fixture)
			sess, err := NewSession(signer)
			if err != nil {
				t.Fatalf("NewSession: %v", err)
			}
			if sess == nil {
				t.Fatal("NewSession returned nil session")
			}
			if sess.encKey == nil {
				t.Error("session encKey is nil")
			}
			if len(sess.encKey) != 32 {
				t.Errorf("expected 32-byte key, got %d bytes", len(sess.encKey))
			}
		})
	}
}

func TestNewSession_UnsupportedKey(t *testing.T) {
	_, err := NewSession(unsupportedSigner{})
	if err == nil {
		t.Fatal("expected error for unsupported key type, got nil")
	}
}

// ── Decrypt ───────────────────────────────────────────────────────────────────

func TestDecrypt_RoundTrip(t *testing.T) {
	fixtures := []struct {
		name    string
		fixture string
	}{
		{"ed25519", "ed25519"},
		{"ecdsa", "ecdsa"},
		{"rsa", "rsa"},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			signer := signerFromFixture(t, tc.fixture)
			sess, err := NewSession(signer)
			if err != nil {
				t.Fatalf("NewSession: %v", err)
			}

			want := "my secret value"
			encoded, err := encryption.Encrypt(sess.encKey, []byte(want))
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}

			got, err := sess.Decrypt(encoded)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if got != want {
				t.Errorf("round-trip: got %q, want %q", got, want)
			}
		})
	}
}

func TestDecrypt_NilKey_ErrNotAuthenticated(t *testing.T) {
	sess := &Session{} // zero value: encKey == nil
	_, err := sess.Decrypt("anything")
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestDecrypt_TamperedCiphertext_WrapsErrDecrypt(t *testing.T) {
	signer := signerFromFixture(t, "ed25519")
	sess, err := NewSession(signer)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	_, err = sess.Decrypt("this-is-not-valid-base64!!!")
	if !errors.Is(err, ErrDecrypt) {
		t.Errorf("expected ErrDecrypt, got %v", err)
	}
}
