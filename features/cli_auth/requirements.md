# requirements.md — cli_auth

## Goal

Enable the CLI to authenticate against the EKVS server by loading an SSH
private key from disk and signing every outbound HTTP request with the
three headers (`X-Timestamp`, `X-Fingerprint`, `X-Signature`) expected by
the server's `AuthMiddleware`.

---

## Scope

### In scope
- `internal/cli/auth.go`: `LoadIdentity` function that parses the private key
  and handles the passphrase flow.
- `internal/cli/signer.go`: `SignedHeaders` function (reusable by `cli_export`
  and `cli_exec`) that produces the three auth headers for a given method/path.
- Passphrase resolution order:
  1. `--passphrase` flag (already declared as persistent flag on `rootCmd`)
  2. `EKVS_PASSPHRASE` environment variable
  3. Prompt on `stdin` (blocking, echoing disabled via `golang.org/x/term`)
- Unit tests in `internal/cli/auth_test.go`.

### Out of scope
- HTTP transport / actual network calls (those belong to `cli_export` / `cli_exec`).
- Session caching across invocations (the CLI is stateless).
- Encryption/decryption of secret values (belongs to `cli_encryption`).

---

## Functional Requirements

### FR-1 `LoadIdentity(identityPath, passphrase string) (crypto.Signer, gossh.PublicKey, string, error)`
- Reads the PEM file at `identityPath`.
- Calls `internal/ssh.ParsePrivateKey`. If it returns `ErrPassphraseRequired`:
  - If `passphrase` arg is non-empty → calls `ParsePrivateKeyWithPassphrase`.
  - Otherwise reads passphrase from `EKVS_PASSPHRASE` env var; if still empty,
    prompts the user on stderr (`"Enter passphrase for <path>: "`) and reads
    from stdin with echo disabled via `golang.org/x/term.ReadPassword`.
- Returns `(signer, pubKey, fingerprint, nil)` on success.
- Returns a wrapped error on any failure (file not found, bad passphrase, etc.).

### FR-2 `SignedHeaders(signer crypto.Signer, fingerprint, method, path string, now time.Time) (map[string]string, error)`
- Delegates to `internal/ssh.CanonicalRequest` + `internal/ssh.Sign`.
- Returns the map `{"X-Timestamp": ..., "X-Fingerprint": ..., "X-Signature": ...}`.
- Pure function; no I/O.

### FR-3 Persistent `--passphrase` flag on `rootCmd`
- Declare `--passphrase` as a persistent string flag (default `""`).
- Also read from `EKVS_PASSPHRASE` env var in `PersistentPreRunE` (same
  pattern as `--server` / `--identity`). Passphrase resolution is:
  flag → env → stdin prompt (done inside `LoadIdentity`, not in `PersistentPreRunE`).

### FR-4 Unit tests (`internal/cli/auth_test.go`)
Table-driven tests using the ed25519 test key from `internal/ssh/testdata/`:
- Unencrypted key loads successfully with empty passphrase.
- Correct fingerprint is returned.
- `SignedHeaders` produces headers with keys `X-Timestamp`, `X-Fingerprint`,
  `X-Signature`.
- `SignedHeaders` output is verifiable with `internal/ssh.Verify` (round-trip).
- Loading a non-existent file returns an error.

---

## Decisions

| # | Decision |
|---|----------|
| D1 | The CLI is stateless: `LoadIdentity` is called on every command invocation. |
| D2 | `SignedHeaders` does **not** accept a `session.Session`; it works directly with `crypto.Signer` to avoid a TUI dependency. |
| D3 | Passphrase resolution order: flag → `EKVS_PASSPHRASE` env → stdin prompt. |
| D4 | `golang.org/x/term` is used for echo-suppressed stdin reading (already an indirect dep via x/crypto; add directly). |

