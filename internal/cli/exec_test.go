package cli_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"ekvs/internal/cli"
	"ekvs/internal/encryption"
)

// TestMain intercepts subprocess invocations used by exec_test scenarios.
// When EKVS_EXEC_HELPER is set, the binary runs as a helper process instead
// of executing the test suite.
func TestMain(m *testing.M) {
	switch os.Getenv("EKVS_EXEC_HELPER") {
	case "print":
		outFile := os.Getenv("EKVS_TEST_OUTFILE")
		_ = os.WriteFile(outFile, []byte(strings.Join(os.Environ(), "\n")), 0o600)
		os.Exit(0)
	case "fail":
		os.Exit(1)
	case "args":
		outFile := os.Getenv("EKVS_TEST_OUTFILE")
		_ = os.WriteFile(outFile, []byte(strings.Join(os.Args, "\n")), 0o600)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// ── helpers ───────────────────────────────────────────────────────────────────

func testBinary(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return exe
}

func runExec(t *testing.T, addr, identity string, extraArgs ...string) error {
	t.Helper()
	cmdArgs := append([]string{"--server", addr, "--identity", identity, "exec"}, extraArgs...)
	return cli.ExecuteWithArgs(cmdArgs, &bytes.Buffer{}, &bytes.Buffer{})
}

func prepareOutfile(t *testing.T) string {
	t.Helper()
	f := t.TempDir() + "/helper_out.txt"
	t.Setenv("EKVS_TEST_OUTFILE", f)
	return f
}

func envFromFile(t *testing.T, path string) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read outfile %s: %v", path, err)
	}
	result := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok {
			result[k] = v
		}
	}
	return result
}

// encryptWithFreshKey generates a new ed25519 key and encrypts plaintext with it.
// Used to produce a blob that cannot be decrypted by the test fixture key.
func encryptWithFreshKey(t *testing.T, plaintext string) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	key, err := encryption.DeriveKey(priv)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	blob, err := encryption.Encrypt(key, []byte(plaintext))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return blob
}

func skipWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("helper-process pattern not supported on Windows")
	}
}

func isExitError(err error) bool {
	var e *exec.ExitError
	return errors.As(err, &e)
}

// ── all secrets ───────────────────────────────────────────────────────────────

func TestExec_AllSecrets_HappyPath(t *testing.T) {
	skipWindows(t)
	signer := loadTestEd25519(t)
	blob1 := encryptForSigner(t, signer, "secret1")
	blob2 := encryptForSigner(t, signer, "secret2")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"secrets": []map[string]string{
				{"key": "KEY1", "value": blob1},
				{"key": "KEY2", "value": blob2},
			},
		})
	}))
	defer ts.Close()

	outFile := prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := envFromFile(t, outFile)
	if env["KEY1"] != "secret1" {
		t.Errorf("KEY1 = %q, want %q", env["KEY1"], "secret1")
	}
	if env["KEY2"] != "secret2" {
		t.Errorf("KEY2 = %q, want %q", env["KEY2"], "secret2")
	}
}

func TestExec_AllSecrets_Empty(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}}) //nolint:errcheck
	}))
	defer ts.Close()

	outFile := prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t)); err != nil {
		t.Fatalf("unexpected error for empty secrets: %v", err)
	}
	raw, _ := os.ReadFile(outFile)
	if strings.Contains(string(raw), "\nKEY1=") || strings.HasPrefix(string(raw), "KEY1=") {
		t.Error("expected no KEY1 in env")
	}
}

// ── single secret ─────────────────────────────────────────────────────────────

func TestExec_SingleKey_HappyPath(t *testing.T) {
	skipWindows(t)
	signer := loadTestEd25519(t)
	blob := encryptForSigner(t, signer, "supersecret")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"key": "MYKEY", "value": blob}) //nolint:errcheck
	}))
	defer ts.Close()

	outFile := prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "MYKEY", "--", testBinary(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := envFromFile(t, outFile)
	if env["MYKEY"] != "supersecret" {
		t.Errorf("MYKEY = %q, want %q", env["MYKEY"], "supersecret")
	}
}

// ── error paths ───────────────────────────────────────────────────────────────

func TestExec_ProjectNotFound(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t))
	if err == nil {
		t.Fatal("expected error for 404 project")
	}
	if !strings.Contains(err.Error(), "project not found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "project not found")
	}
}

func TestExec_KeyNotFound(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer ts.Close()

	prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "NO_SUCH_KEY", "--", testBinary(t))
	if err == nil {
		t.Fatal("expected error for 404 key")
	}
	if !strings.Contains(err.Error(), "project or key not found") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "project or key not found")
	}
}

func TestExec_ServerError(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()

	prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t)); err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestExec_DecryptFailure(t *testing.T) {
	skipWindows(t)
	blob := encryptWithFreshKey(t, "wrongkey")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"secrets": []map[string]string{{"key": "BAD", "value": blob}},
		})
	}))
	defer ts.Close()

	prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t)); err == nil {
		t.Fatal("expected decrypt error")
	}
}

func TestExec_ProgramNotFound(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}}) //nolint:errcheck
	}))
	defer ts.Close()

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", "/no/such/binary/ekvs_nonexistent"); err == nil {
		t.Fatal("expected error for non-existent binary")
	}
}

func TestExec_ProgramExitsNonZero(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}}) //nolint:errcheck
	}))
	defer ts.Close()

	prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "fail")

	err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t))
	if err == nil {
		t.Fatal("expected non-zero exit error")
	}
	if !isExitError(err) {
		t.Errorf("expected *exec.ExitError, got %T: %v", err, err)
	}
}

func TestExec_ProgramReceivesArgs(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}}) //nolint:errcheck
	}))
	defer ts.Close()

	outFile := prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "args")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t), "argA", "argB"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, _ := os.ReadFile(outFile)
	content := string(raw)
	if !strings.Contains(content, "argA") || !strings.Contains(content, "argB") {
		t.Errorf("subprocess args = %q; want argA and argB", content)
	}
}

func TestExec_InheritsParentEnv(t *testing.T) {
	skipWindows(t)
	signer := loadTestEd25519(t)
	blob := encryptForSigner(t, signer, "injected")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"secrets": []map[string]string{{"key": "INJECTED_KEY", "value": blob}},
		})
	}))
	defer ts.Close()

	outFile := prepareOutfile(t)
	t.Setenv("EKVS_EXEC_HELPER", "print")
	t.Setenv("PARENT_ONLY_VAR", "parent_value")

	if err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "proj", "--", testBinary(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	env := envFromFile(t, outFile)
	if env["PARENT_ONLY_VAR"] != "parent_value" {
		t.Errorf("PARENT_ONLY_VAR = %q, want %q", env["PARENT_ONLY_VAR"], "parent_value")
	}
	if env["INJECTED_KEY"] != "injected" {
		t.Errorf("INJECTED_KEY = %q, want %q", env["INJECTED_KEY"], "injected")
	}
}

func TestExec_TooFewArgs(t *testing.T) {
	err := cli.ExecuteWithArgs(
		[]string{"--server", "localhost:9090", "--identity", exportFixturePath("ed25519"), "exec"},
		&bytes.Buffer{}, &bytes.Buffer{},
	)
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestExec_MissingDoubleDash(t *testing.T) {
	skipWindows(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"secrets": []any{}}) //nolint:errcheck
	}))
	defer ts.Close()

	// "proj" + "env" without "--" → ArgsLenAtDash returns -1 → error.
	err := runExec(t, serverAddr(ts), exportFixturePath("ed25519"), "exec", "proj", "env")
	if err == nil {
		t.Fatal("expected error for missing -- separator")
	}
}
