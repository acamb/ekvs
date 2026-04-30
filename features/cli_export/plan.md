# plan.md — cli_export

Ordered list of tasks to implement the `cli_export` feature.

---

## Tasks

### 1. Create `internal/cli/client.go`

**File**: `internal/cli/client.go`

- Define `secretEntry` struct with `Key string` and `Value string` (JSON tags `"key"`, `"value"`).
- Define `ErrNotFound` sentinel error.
- Define `Client` struct with `BaseURL string` and `HTTPClient *http.Client`.
- Implement `NewClient(baseURL string) *Client` — sets `HTTPClient` to `http.DefaultClient`.
- Implement `(c *Client) ListSecrets(signer crypto.Signer, fingerprint, project string) ([]secretEntry, error)`:
  - Build path `/projects/{project}/secrets`.
  - Call `SignedHeaders(signer, fingerprint, "GET", path, time.Now())`.
  - Execute `GET {BaseURL}{path}` with the signed headers.
  - Return `ErrNotFound` on HTTP 404.
  - JSON-decode `{"secrets": [...]}` into `[]secretEntry`.
- Implement `(c *Client) GetSecret(signer crypto.Signer, fingerprint, project, key string) (*secretEntry, error)`:
  - Build path `/projects/{project}/secrets/{key}`.
  - Same signing + HTTP pattern as `ListSecrets`.
  - Return `ErrNotFound` on HTTP 404.
  - JSON-decode `{"key": "...", "value": "..."}` into `secretEntry`.

---

### 2. Implement `internal/cli/export.go`

**File**: `internal/cli/export.go` (replace stub)

Replace the `RunE` stub with the real implementation:

- Call `LoadIdentity(flagIdentity, flagPassphrase)` → `signer, _, fingerprint, err`.
- Call `NewSession(signer)` → `session, err`.
- Instantiate `NewClient("http://" + flagServer)`.
- **Single key** (`len(args) == 2`):
  - Call `client.GetSecret(signer, fingerprint, args[0], args[1])`.
  - Decrypt the value via `session.Decrypt(entry.Value)`.
  - Write `KEY=plaintext\n` to `cmd.OutOrStdout()`.
- **All secrets** (`len(args) == 1`):
  - Call `client.ListSecrets(signer, fingerprint, args[0])`.
  - For each entry, decrypt via `session.Decrypt(entry.Value)`.
  - Write `KEY=plaintext\n` for each entry to `cmd.OutOrStdout()` in API-returned order.
- Wrap `ErrNotFound` into a user-visible error message (e.g. `"project or key not found"`).

---

### 3. Write unit tests in `internal/cli/export_test.go`

**File**: `internal/cli/export_test.go`

Table-driven tests using `httptest.NewServer` and `ExecuteWithArgs`:

- **Export single key — happy path**: server returns a valid encrypted blob; assert `KEY=plaintext\n` on stdout.
- **Export single key — 404**: server returns 404; assert non-nil error.
- **Export all secrets — happy path**: server returns multiple entries; assert all `KEY=value` lines printed in order.
- **Export all secrets — empty list**: server returns `{"secrets": []}` → no output, no error.
- **Export all secrets — 404**: project not found; assert non-nil error.
- **Encryption round-trip**: pre-encrypt a fixture value with `encryption.Encrypt` + `encryption.DeriveKey` using an ed25519 key; serve that blob; assert plaintext is recovered.

Use `ExecuteWithArgs` with `--server <testServerHostPort>` and `--identity <path-to-test-ed25519-key>`.

---

### 4. Verify no regression

- Confirm `exec.go` stub compiles unchanged after adding `client.go` and updating `export.go`.
- Run `go build ./...` and `go test ./...` — all must pass.

