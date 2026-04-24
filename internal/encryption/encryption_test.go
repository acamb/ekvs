package encryption

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	internalssh "ekvs/internal/ssh"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// testdataPath returns the absolute path to internal/ssh/testdata/<name>.
// It navigates from the current package directory up two levels.
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

// ── DeriveKey ─────────────────────────────────────────────────────────────────

func TestDeriveKey_HappyPath(t *testing.T) {
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
			key, err := DeriveKey(signer)
			if err != nil {
				t.Fatalf("DeriveKey: %v", err)
			}
			if len(key) != 32 {
				t.Errorf("expected 32-byte key, got %d bytes", len(key))
			}
		})
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	fixtures := []string{"ed25519", "ecdsa", "rsa"}

	for _, f := range fixtures {
		t.Run(f, func(t *testing.T) {
			signer := signerFromFixture(t, f)

			key1, err := DeriveKey(signer)
			if err != nil {
				t.Fatalf("first DeriveKey: %v", err)
			}
			key2, err := DeriveKey(signer)
			if err != nil {
				t.Fatalf("second DeriveKey: %v", err)
			}
			if !bytes.Equal(key1, key2) {
				t.Error("DeriveKey is not deterministic: two calls returned different results")
			}
		})
	}
}

func TestDeriveKey_UnsupportedType(t *testing.T) {
	s := unsupportedSigner{}
	_, err := DeriveKey(s)
	if !errors.Is(err, ErrUnsupportedKeyType) {
		t.Errorf("expected ErrUnsupportedKeyType, got %v", err)
	}
}

func TestDeriveKey_RSATooSmall(t *testing.T) {
	// Generate an RSA-1024 key (insecure, below the 2048-bit minimum).
	smallKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generating small RSA key: %v", err)
	}
	_, err = DeriveKey(smallKey)
	if !errors.Is(err, ErrUnsupportedKeyType) {
		t.Errorf("expected ErrUnsupportedKeyType for RSA-1024, got %v", err)
	}
}

// unsupportedSigner satisfies crypto.Signer with an unknown public key type.
type unsupportedSigner struct{}

func (unsupportedSigner) Public() crypto.PublicKey { return struct{}{} }
func (unsupportedSigner) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	return nil, nil
}

// ── Encrypt ───────────────────────────────────────────────────────────────────

func TestEncrypt_HappyPath(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"non-empty plaintext", []byte("hello, world!")},
		{"empty plaintext", []byte{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Encrypt(key, tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			if got == "" {
				t.Error("Encrypt returned empty string")
			}
		})
	}
}

func TestEncrypt_NonceRandomness(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	plaintext := []byte("deterministic input")

	ct1, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("first Encrypt: %v", err)
	}
	ct2, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("second Encrypt: %v", err)
	}
	if ct1 == ct2 {
		t.Error("Encrypt produced identical ciphertexts for the same input (nonce not random)")
	}
}

// ── Decrypt ───────────────────────────────────────────────────────────────────

func TestDecrypt_WrongKey(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	encoded, err := Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	wrongKey := make([]byte, 32)
	if _, err := rand.Read(wrongKey); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	_, err = Decrypt(wrongKey, encoded)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	// blob of 8 bytes < nonceSize (12)
	short := base64.StdEncoding.EncodeToString(make([]byte, 8))
	_, err := Decrypt(make([]byte, 32), short)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecrypt_TamperedTag(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	encoded, err := Encrypt(key, []byte("original"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	blob, _ := base64.StdEncoding.DecodeString(encoded)
	blob[len(blob)-1] ^= 0xFF // flip last byte of tag
	tampered := base64.StdEncoding.EncodeToString(blob)

	_, err = Decrypt(key, tampered)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := Decrypt(make([]byte, 32), "not-base64!!")
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

func TestEncrypt_BadKeySize(t *testing.T) {
	// AES requires 16, 24, or 32-byte keys; a 7-byte key must fail.
	_, err := Encrypt(make([]byte, 7), []byte("hello"))
	if err == nil {
		t.Fatal("expected error for bad key size, got nil")
	}
}

func TestDecrypt_BadKeySize(t *testing.T) {
	// First produce a valid ciphertext with a proper key.
	validKey := make([]byte, 32)
	if _, err := rand.Read(validKey); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	encoded, err := Encrypt(validKey, []byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Now try to decrypt with a bad-size key.
	_, err = Decrypt(make([]byte, 7), encoded)
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected ErrDecryptionFailed, got %v", err)
	}
}

// ── Round-trip ────────────────────────────────────────────────────────────────

func TestRoundTrip(t *testing.T) {
	fixtures := []struct {
		name    string
		fixture string
	}{
		{"ed25519", "ed25519"},
		{"ecdsa", "ecdsa"},
		{"rsa", "rsa"},
	}

	plaintexts := [][]byte{
		[]byte("first secret"),
		[]byte("second secret"),
		[]byte("third secret"),
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			signer := signerFromFixture(t, tc.fixture)
			key, err := DeriveKey(signer)
			if err != nil {
				t.Fatalf("DeriveKey: %v", err)
			}

			// RT-4: encrypt multiple values and decrypt independently
			encoded := make([]string, len(plaintexts))
			for i, pt := range plaintexts {
				enc, err := Encrypt(key, pt)
				if err != nil {
					t.Fatalf("Encrypt[%d]: %v", i, err)
				}
				encoded[i] = enc
			}

			for i, pt := range plaintexts {
				got, err := Decrypt(key, encoded[i])
				if err != nil {
					t.Fatalf("Decrypt[%d]: %v", i, err)
				}
				if !bytes.Equal(got, pt) {
					t.Errorf("round-trip[%d]: got %q, want %q", i, got, pt)
				}
			}
		})
	}
}

func TestRoundTrip_EmptyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	encoded, err := Encrypt(key, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	got, err := Decrypt(key, encoded)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty plaintext, got %q", got)
	}
}
