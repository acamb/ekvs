package cli_test

import (
	"bytes"
	"crypto"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"ekvs/internal/cli"
	"ekvs/internal/encryption"
	internalssh "ekvs/internal/ssh"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func exportFixturePath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "ssh", "testdata", name)
}

func loadTestEd25519(t *testing.T) crypto.Signer {
	t.Helper()
	pem, err := os.ReadFile(exportFixturePath("ed25519"))
	if err != nil {
		t.Fatalf("read ed25519 fixture: %v", err)
	}
	signer, _, err := internalssh.ParsePrivateKey(pem)
	if err != nil {
		t.Fatalf("parse ed25519: %v", err)
	}
	return signer
}

func encryptForSigner(t *testing.T, signer crypto.Signer, plaintext string) string {
	t.Helper()
	key, err := encryption.DeriveKey(signer)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	encoded, err := encryption.Encrypt(key, []byte(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return encoded
}

// serverAddr strips the scheme prefix from httptest.Server.URL.
func serverAddr(ts *httptest.Server) string {
	return strings.TrimPrefix(ts.URL, "http://")
}

// runExport invokes "ekvs export <extraArgs...>" and returns stdout + error.
func runExport(t *testing.T, addr, identity string, extraArgs ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmdArgs := append([]string{"--server", addr, "--identity", identity, "export"}, extraArgs...)
	err := cli.ExecuteWithArgs(cmdArgs, &out, &bytes.Buffer{})
	return out.String(), err
}

// ── export all secrets ────────────────────────────────────────────────────────

func TestExport_AllSecrets_HappyPath(t *testing.T) {
	signer := loadTestEd25519(t)
	blob1 := encryptForSigner(t, signer, "val1")
	blob2 := encryptForSigner(t, signer, "val2")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"secrets": []map[string]string{
				{"key": "KEY1", "value": blob1},
				{"key": "KEY2", "value": blob2},
			},
		})
	}))
	defer ts.Close()

	out, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "KEY1=val1\nKEY2=val2\n"
	if out != want {
		t.Errorf("output = %q, want %q", out, want)
	}
}

func TestExport_AllSecrets_Empty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}}) //nolint:errcheck
	}))
	defer ts.Close()

	out, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestExport_AllSecrets_ProjectNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	_, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj")
	if err == nil {
		t.Fatal("expected error for 404 project")
	}
}

func TestExport_AllSecrets_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	_, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

// ── export single secret ──────────────────────────────────────────────────────

func TestExport_SingleSecret_HappyPath(t *testing.T) {
	signer := loadTestEd25519(t)
	blob := encryptForSigner(t, signer, "secret123")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"key": "MYKEY", "value": blob}) //nolint:errcheck
	}))
	defer ts.Close()

	out, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "MYKEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "MYKEY=secret123\n" {
		t.Errorf("output = %q, want %q", out, "MYKEY=secret123\n")
	}
}

func TestExport_SingleSecret_KeyNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	_, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "MYKEY")
	if err == nil {
		t.Fatal("expected error for 404 key")
	}
}

// ── encryption round-trip ─────────────────────────────────────────────────────

func TestExport_EncryptionRoundTrip(t *testing.T) {
	signer := loadTestEd25519(t)
	plaintext := "super-secret-value-42"
	blob := encryptForSigner(t, signer, plaintext)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"secrets": []map[string]string{{"key": "ROUND_TRIP", "value": blob}},
		})
	}))
	defer ts.Close()

	out, err := runExport(t, serverAddr(ts), exportFixturePath("ed25519"), "proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "ROUND_TRIP=" + plaintext + "\n"
	if out != want {
		t.Errorf("round-trip output = %q, want %q", out, want)
	}
}
