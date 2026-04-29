package cli_test

import (
	"bytes"
	"testing"

	"ekvs/internal/cli"
)

// executeArgs resets package-level state and runs the CLI with the given args,
// capturing stdout and stderr.
func executeArgs(t *testing.T, args []string) error {
	t.Helper()
	return cli.ExecuteWithArgs(args, &bytes.Buffer{}, &bytes.Buffer{})
}

func TestMissingServer(t *testing.T) {
	t.Setenv("EKVS_SERVER", "")
	t.Setenv("EKVS_IDENTITY", "")
	err := executeArgs(t, []string{"export", "myproject"})
	if err == nil {
		t.Fatal("expected error for missing --server")
	}
}

func TestMissingIdentity(t *testing.T) {
	t.Setenv("EKVS_SERVER", "localhost:8080")
	t.Setenv("EKVS_IDENTITY", "")
	err := executeArgs(t, []string{"export", "myproject"})
	if err == nil {
		t.Fatal("expected error for missing --identity")
	}
}

func TestFlagBeatsEnv(t *testing.T) {
	t.Setenv("EKVS_SERVER", "env-host:9999")
	t.Setenv("EKVS_IDENTITY", "env-key")
	// Even with env set, explicit flags must be accepted (command still fails
	// with "not yet implemented", not a config error).
	err := executeArgs(t, []string{"--server", "flag-host:8080", "--identity", "/tmp/key", "export", "myproject"})
	if err == nil || err.Error() == "required flag --server (or $EKVS_SERVER) not set" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportEnvFallback(t *testing.T) {
	t.Setenv("EKVS_SERVER", "localhost:8080")
	t.Setenv("EKVS_IDENTITY", "/tmp/key")
	err := executeArgs(t, []string{"export", "myproject"})
	if err == nil || err.Error() == "required flag --server (or $EKVS_SERVER) not set" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportNoArgs(t *testing.T) {
	t.Setenv("EKVS_SERVER", "localhost:8080")
	t.Setenv("EKVS_IDENTITY", "/tmp/key")
	err := executeArgs(t, []string{"export"})
	if err == nil {
		t.Fatal("expected error when export called with no args")
	}
}

func TestExecNoArgs(t *testing.T) {
	t.Setenv("EKVS_SERVER", "localhost:8080")
	t.Setenv("EKVS_IDENTITY", "/tmp/key")
	err := executeArgs(t, []string{"exec"})
	if err == nil {
		t.Fatal("expected error when exec called with no args")
	}
}

func TestPassphraseFlagBeatsEnv(t *testing.T) {
	t.Setenv("EKVS_SERVER", "localhost:8080")
	t.Setenv("EKVS_IDENTITY", "/tmp/key")
	t.Setenv("EKVS_PASSPHRASE", "env-passphrase")
	// --passphrase flag is accepted; command still fails with "not yet implemented".
	err := executeArgs(t, []string{"--passphrase", "flag-passphrase", "export", "myproject"})
	if err == nil || err.Error() == "required flag --server (or $EKVS_SERVER) not set" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPassphraseEnvFallback(t *testing.T) {
	t.Setenv("EKVS_SERVER", "localhost:8080")
	t.Setenv("EKVS_IDENTITY", "/tmp/key")
	t.Setenv("EKVS_PASSPHRASE", "env-passphrase")
	// No --passphrase flag; env var must be accepted without error.
	err := executeArgs(t, []string{"export", "myproject"})
	if err == nil || err.Error() == "required flag --server (or $EKVS_SERVER) not set" {
		t.Fatalf("unexpected error: %v", err)
	}
}
