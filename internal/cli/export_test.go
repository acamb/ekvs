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

// ── --output flag ─────────────────────────────────────────────────────────────

// runExportToFile invokes "ekvs export ... --output <outPath>" and returns the error.
func runExportToFile(t *testing.T, addr, identity, outPath string, extraArgs ...string) error {
	t.Helper()
	cmdArgs := append([]string{"--server", addr, "--identity", identity, "export"}, extraArgs...)
	cmdArgs = append(cmdArgs, "--output", outPath)
	return cli.ExecuteWithArgs(cmdArgs, &bytes.Buffer{}, &bytes.Buffer{})
}

func TestExport_OutputFile_HappyPath(t *testing.T) {
	signer := loadTestEd25519(t)
	blob := encryptForSigner(t, signer, "hunter2")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"key": "DB_PASS", "value": blob}) //nolint:errcheck
	}))
	defer ts.Close()

	out := filepath.Join(t.TempDir(), "secret.txt")
	err := runExportToFile(t, serverAddr(ts), exportFixturePath("ed25519"), out, "proj", "DB_PASS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatalf("read output file: %v", readErr)
	}
	if string(raw) != "hunter2" {
		t.Errorf("file content = %q, want %q", string(raw), "hunter2")
	}
}

func TestExport_OutputFile_Overwrite(t *testing.T) {
	signer := loadTestEd25519(t)
	blob := encryptForSigner(t, signer, "newvalue")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"key": "KEY", "value": blob}) //nolint:errcheck
	}))
	defer ts.Close()

	out := filepath.Join(t.TempDir(), "secret.txt")
	// Pre-create file with different content.
	if err := os.WriteFile(out, []byte("oldvalue"), 0600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := runExportToFile(t, serverAddr(ts), exportFixturePath("ed25519"), out, "proj", "KEY"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, _ := os.ReadFile(out)
	if string(raw) != "newvalue" {
		t.Errorf("file content = %q, want %q", string(raw), "newvalue")
	}
}

func TestExport_OutputFile_MissingKeyName(t *testing.T) {
	// Server should never be hit — guard fires first.
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	out := filepath.Join(t.TempDir(), "secret.txt")
	err := runExportToFile(t, serverAddr(ts), exportFixturePath("ed25519"), out, "proj") // no keyName
	if err == nil {
		t.Fatal("expected error for --output without keyName")
	}
	if !strings.Contains(err.Error(), "--output requires a keyName argument") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "--output requires a keyName argument")
	}
	if called {
		t.Error("server was called but should not have been")
	}
}

func TestExport_OutputFile_DirectoryNotFound(t *testing.T) {
	signer := loadTestEd25519(t)
	blob := encryptForSigner(t, signer, "value")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"key": "K", "value": blob}) //nolint:errcheck
	}))
	defer ts.Close()

	out := filepath.Join(t.TempDir(), "nonexistent", "secret.txt")
	err := runExportToFile(t, serverAddr(ts), exportFixturePath("ed25519"), out, "proj", "K")
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
	if !strings.Contains(err.Error(), "write output file") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "write output file")
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Error("output file should not have been created")
	}
}

func TestExport_OutputFile_DecryptFailure(t *testing.T) {
	// Encrypt with a key that does NOT match the test fixture → decrypt will fail.
	signer := loadTestEd25519(t)
	key, err := encryption.DeriveKey(signer)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	// Corrupt the blob so AES-GCM authentication fails.
	blob, err := encryption.Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// Flip a byte in the base64 to corrupt the ciphertext.
	corrupted := []byte(blob)
	corrupted[len(corrupted)-1] ^= 0xFF
	corruptedBlob := string(corrupted)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"key": "K", "value": corruptedBlob}) //nolint:errcheck
	}))
	defer ts.Close()

	out := filepath.Join(t.TempDir(), "secret.txt")
	if err := runExportToFile(t, serverAddr(ts), exportFixturePath("ed25519"), out, "proj", "K"); err == nil {
		t.Fatal("expected decrypt error")
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Error("output file should not have been created on decrypt failure")
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
