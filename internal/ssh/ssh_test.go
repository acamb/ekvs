package ssh

import (
	"crypto"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func readFile(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("reading testdata/%s: %v", name, err)
	}
	return b
}

// ── ParsePrivateKey ───────────────────────────────────────────────────────────

func TestParsePrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		wantErr error
	}{
		{name: "ed25519", fixture: "ed25519"},
		{name: "ecdsa", fixture: "ecdsa"},
		{name: "rsa", fixture: "rsa"},
		{name: "invalid PEM", fixture: "", wantErr: errors.New("any")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var pemBytes []byte
			if tc.fixture != "" {
				pemBytes = readFile(t, tc.fixture)
			} else {
				pemBytes = []byte("not a pem block")
			}

			signer, pub, err := ParsePrivateKey(pemBytes)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if signer == nil {
				t.Error("signer is nil")
			}
			if pub == nil {
				t.Error("public key is nil")
			}
		})
	}
}

func TestParsePrivateKey_UnsupportedType(t *testing.T) {
	// Build a PEM block with an unsupported type. golang.org/x/crypto/ssh
	// ParseRawPrivateKey will return an error for an unknown header.
	unsupportedPEM := []byte("-----BEGIN UNSUPPORTED KEY TYPE-----\nYWJj\n-----END UNSUPPORTED KEY TYPE-----\n")
	_, _, err := ParsePrivateKey(unsupportedPEM)
	if err == nil {
		t.Fatal("expected error for unsupported PEM type, got nil")
	}
}

func TestParsePrivateKey_PassphraseRequired(t *testing.T) {
	for _, fixture := range []string{"ed25519-passphrase", "rsa-passphrase"} {
		t.Run(fixture, func(t *testing.T) {
			pemBytes := readFile(t, fixture)
			_, _, err := ParsePrivateKey(pemBytes)
			if !errors.Is(err, ErrPassphraseRequired) {
				t.Fatalf("expected ErrPassphraseRequired, got %v", err)
			}
		})
	}
}

func TestParsePrivateKeyWithPassphrase(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		passphrase string
		wantErr    bool
	}{
		{name: "ed25519 correct passphrase", fixture: "ed25519-passphrase", passphrase: "testpass"},
		{name: "rsa correct passphrase", fixture: "rsa-passphrase", passphrase: "testpass"},
		{name: "ed25519 wrong passphrase", fixture: "ed25519-passphrase", passphrase: "wrongpass", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pemBytes := readFile(t, tc.fixture)
			signer, pub, err := ParsePrivateKeyWithPassphrase(pemBytes, []byte(tc.passphrase))
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if signer == nil || pub == nil {
				t.Fatal("expected signer and public key, got nil")
			}
			// Round-trip: sign + verify
			msg := []byte("round-trip test message")
			sigBlob, err := Sign(signer, msg)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}
			if err := Verify(pub, msg, sigBlob); err != nil {
				t.Fatalf("Verify: %v", err)
			}
		})
	}
}

// ── ParseAuthorizedKey ────────────────────────────────────────────────────────

func TestParseAuthorizedKey(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		wantErr bool
	}{
		{name: "ed25519", fixture: "ed25519.pub"},
		{name: "ecdsa", fixture: "ecdsa.pub"},
		{name: "rsa", fixture: "rsa.pub"},
		{name: "malformed", fixture: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var line []byte
			if tc.fixture != "" {
				line = readFile(t, tc.fixture)
			} else {
				line = []byte("this is not a valid authorized_keys line")
			}

			pub, err := ParseAuthorizedKey(line)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pub == nil {
				t.Error("public key is nil")
			}
		})
	}
}

// ── Fingerprint ───────────────────────────────────────────────────────────────

func TestFingerprint(t *testing.T) {
	fixtures := []string{"ed25519.pub", "ecdsa.pub", "rsa.pub"}

	for _, f := range fixtures {
		t.Run(f, func(t *testing.T) {
			line := readFile(t, f)
			pub, err := ParseAuthorizedKey(line)
			if err != nil {
				t.Fatalf("ParseAuthorizedKey: %v", err)
			}

			fp := Fingerprint(pub)
			if !strings.HasPrefix(fp, "SHA256:") {
				t.Errorf("fingerprint %q does not start with SHA256:", fp)
			}
			if len(fp) < 10 {
				t.Errorf("fingerprint %q seems too short", fp)
			}
		})
	}
}

// ── CanonicalRequest ──────────────────────────────────────────────────────────

func TestCanonicalRequest(t *testing.T) {
	ts := time.Unix(1745481600, 0).UTC()

	tests := []struct {
		method string
		path   string
		ts     time.Time
		want   string
	}{
		{
			method: "GET",
			path:   "/projects/myproject",
			ts:     ts,
			want:   "GET\n/projects/myproject\n1745481600",
		},
		{
			method: "put", // should be uppercased
			path:   "/secrets/foo",
			ts:     ts,
			want:   "PUT\n/secrets/foo\n1745481600",
		},
		{
			method: "DELETE",
			path:   "/projects/x",
			ts:     ts,
			want:   "DELETE\n/projects/x\n1745481600",
		},
	}

	for _, tc := range tests {
		t.Run(tc.method+":"+tc.path, func(t *testing.T) {
			got := CanonicalRequest(tc.method, tc.path, tc.ts)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// ── Sign + Verify round-trip ──────────────────────────────────────────────────

func TestSignVerify_RoundTrip(t *testing.T) {
	fixtures := []struct {
		name    string
		keyFile string
		pubFile string
	}{
		{"ed25519", "ed25519", "ed25519.pub"},
		{"ecdsa", "ecdsa", "ecdsa.pub"},
		{"rsa", "rsa", "rsa.pub"},
	}

	message := []byte(CanonicalRequest("GET", "/projects/test", time.Now()))

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			signer, _, err := ParsePrivateKey(readFile(t, tc.keyFile))
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}
			pub, err := ParseAuthorizedKey(readFile(t, tc.pubFile))
			if err != nil {
				t.Fatalf("ParseAuthorizedKey: %v", err)
			}

			blob, err := Sign(signer, message)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			if err := Verify(pub, message, blob); err != nil {
				t.Fatalf("Verify: %v", err)
			}
		})
	}
}

func TestVerify_TamperedMessage(t *testing.T) {
	fixtures := []struct {
		name    string
		keyFile string
		pubFile string
	}{
		{"ed25519", "ed25519", "ed25519.pub"},
		{"ecdsa", "ecdsa", "ecdsa.pub"},
		{"rsa", "rsa", "rsa.pub"},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			signer, _, err := ParsePrivateKey(readFile(t, tc.keyFile))
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}
			pub, err := ParseAuthorizedKey(readFile(t, tc.pubFile))
			if err != nil {
				t.Fatalf("ParseAuthorizedKey: %v", err)
			}

			original := []byte("GET\n/projects/test\n1745481600")
			blob, err := Sign(signer, original)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			tampered := []byte("PUT\n/projects/test\n1745481600")
			err = Verify(pub, tampered, blob)
			if !errors.Is(err, ErrInvalidSignature) {
				t.Errorf("expected ErrInvalidSignature, got %v", err)
			}
		})
	}
}

func TestVerify_TamperedSignature(t *testing.T) {
	signer, _, err := ParsePrivateKey(readFile(t, "ed25519"))
	if err != nil {
		t.Fatalf("ParsePrivateKey: %v", err)
	}
	pub, err := ParseAuthorizedKey(readFile(t, "ed25519.pub"))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}

	msg := []byte("GET\n/projects/test\n1745481600")
	blob, err := Sign(signer, msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	// Flip a byte in the middle of the blob.
	tampered := make([]byte, len(blob))
	copy(tampered, blob)
	tampered[len(tampered)/2] ^= 0xFF

	err = Verify(pub, msg, tampered)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerify_MalformedBlob(t *testing.T) {
	pub, err := ParseAuthorizedKey(readFile(t, "ed25519.pub"))
	if err != nil {
		t.Fatalf("ParseAuthorizedKey: %v", err)
	}

	msg := []byte("GET\n/projects/test\n1745481600")
	garbage := []byte("this is not a valid ssh.Signature blob")

	err = Verify(pub, msg, garbage)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerify_WrongPublicKey(t *testing.T) {
	fixtures := []struct {
		name    string
		keyFile string
		pubFile string // different key
	}{
		{"ed25519 signed, ecdsa pubkey", "ed25519", "ecdsa.pub"},
		{"rsa signed, ed25519 pubkey", "rsa", "ed25519.pub"},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			signer, _, err := ParsePrivateKey(readFile(t, tc.keyFile))
			if err != nil {
				t.Fatalf("ParsePrivateKey: %v", err)
			}
			wrongPub, err := ParseAuthorizedKey(readFile(t, tc.pubFile))
			if err != nil {
				t.Fatalf("ParseAuthorizedKey: %v", err)
			}

			msg := []byte("GET\n/projects/test\n1745481600")
			blob, err := Sign(signer, msg)
			if err != nil {
				t.Fatalf("Sign: %v", err)
			}

			err = Verify(wrongPub, msg, blob)
			if !errors.Is(err, ErrInvalidSignature) {
				t.Errorf("expected ErrInvalidSignature, got %v", err)
			}
		})
	}
}

// ── Sign error path ───────────────────────────────────────────────────────────

// unsupportedSigner is a crypto.Signer backed by an unknown key type.
// It causes gossh.NewSignerFromSigner to fail because the key type is not
// in the list of types supported by golang.org/x/crypto/ssh.
type unsupportedSigner struct{}

func (unsupportedSigner) Public() crypto.PublicKey { return struct{}{} }
func (unsupportedSigner) Sign(_ io.Reader, _ []byte, _ crypto.SignerOpts) ([]byte, error) {
	return nil, nil
}

func TestSign_UnsupportedSignerType(t *testing.T) {
	_, err := Sign(unsupportedSigner{}, []byte("msg"))
	if err == nil {
		t.Fatal("expected error for unsupported signer, got nil")
	}
}

// ── CheckTimestamp ────────────────────────────────────────────────────────────

func TestCheckTimestamp(t *testing.T) {
	window := 30 * time.Second

	tests := []struct {
		name    string
		delta   time.Duration
		wantErr bool
	}{
		{"zero delta", 0, false},
		{"within window positive", 29 * time.Second, false},
		{"within window negative", -29 * time.Second, false},
		// boundary positive: ts is 30s in the past → |delta| == window → rejected
		{"on boundary positive", 30 * time.Second, true},
		// Note: boundary-negative (future ts exactly 30s ahead) is not tested
		// because the two time.Now() calls inside the test and CheckTimestamp
		// are microseconds apart, making the assertion inherently flaky.
		{"outside window positive", 31 * time.Second, true},
		{"outside window negative", -31 * time.Second, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := time.Now().UTC().Add(-tc.delta)
			err := CheckTimestamp(ts, window)
			if tc.wantErr {
				if !errors.Is(err, ErrReplayDetected) {
					t.Errorf("expected ErrReplayDetected, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
