package cli_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"ekvs/internal/cli"
	internalssh "ekvs/internal/ssh"
)

const testKeyPath = "../../internal/ssh/testdata/ed25519"
const testKeyPathPassphrase = "../../internal/ssh/testdata/ed25519-passphrase"
const testPassphrase = "testpass"

// ── LoadIdentity ──────────────────────────────────────────────────────────────

func TestLoadIdentity_Unencrypted(t *testing.T) {
	signer, pub, fp, err := cli.LoadIdentity(testKeyPath, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signer == nil {
		t.Fatal("signer is nil")
	}
	if pub == nil {
		t.Fatal("pubKey is nil")
	}
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Fatalf("fingerprint %q does not start with SHA256:", fp)
	}
}

func TestLoadIdentity_FileNotFound(t *testing.T) {
	_, _, _, err := cli.LoadIdentity("/nonexistent/path/key", "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "/nonexistent/path/key") {
		t.Fatalf("error should mention path, got: %v", err)
	}
}

func TestLoadIdentity_WithPassphraseFlag(t *testing.T) {
	signer, pub, fp, err := cli.LoadIdentity(testKeyPathPassphrase, testPassphrase)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signer == nil || pub == nil {
		t.Fatal("signer or pubKey is nil")
	}
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Fatalf("fingerprint %q does not start with SHA256:", fp)
	}
}

func TestLoadIdentity_WrongPassphrase(t *testing.T) {
	_, _, _, err := cli.LoadIdentity(testKeyPathPassphrase, "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong passphrase")
	}
}

// ── SignedHeaders ─────────────────────────────────────────────────────────────

func TestSignedHeaders_Keys(t *testing.T) {
	signer, _, fp, err := cli.LoadIdentity(testKeyPath, "")
	if err != nil {
		t.Fatalf("load identity: %v", err)
	}

	headers, err := cli.SignedHeaders(signer, fp, "GET", "/v1/projects", time.Now())
	if err != nil {
		t.Fatalf("SignedHeaders: %v", err)
	}

	for _, key := range []string{"X-Timestamp", "X-Fingerprint", "X-Signature"} {
		if _, ok := headers[key]; !ok {
			t.Errorf("missing header %q", key)
		}
	}
	if len(headers) != 3 {
		t.Errorf("expected exactly 3 headers, got %d", len(headers))
	}
}

func TestSignedHeaders_RoundTrip(t *testing.T) {
	signer, pub, fp, err := cli.LoadIdentity(testKeyPath, "")
	if err != nil {
		t.Fatalf("load identity: %v", err)
	}

	now := time.Now()
	method, path := "GET", "/v1/projects"

	headers, err := cli.SignedHeaders(signer, fp, method, path, now)
	if err != nil {
		t.Fatalf("SignedHeaders: %v", err)
	}

	sigBlob, err := base64.StdEncoding.DecodeString(headers["X-Signature"])
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	canonical := internalssh.CanonicalRequest(method, path, now)
	if err := internalssh.Verify(pub, []byte(canonical), sigBlob); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}
