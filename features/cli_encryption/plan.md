# plan.md — cli_encryption

Ordered list of tasks to implement the `cli_encryption` feature.

---

## Tasks

### 1. Create `internal/cli/session.go`

**File**: `internal/cli/session.go`

- Add package-level error sentinels: `ErrNotAuthenticated`, `ErrDecrypt`.
- Define `Session` struct with unexported `encKey []byte` field.
- Implement `NewSession(signer crypto.Signer) (*Session, error)`:
  - Call `encryption.DeriveKey(signer)`; return error on failure.
  - Return a `*Session` with `encKey` set on success.
- Implement `(s *Session) Decrypt(encoded string) (string, error)`:
  - Return `ErrNotAuthenticated` if `encKey == nil`.
  - Delegate to `encryption.Decrypt(s.encKey, encoded)`; wrap errors in `ErrDecrypt`.

---

### 2. Write unit tests in `internal/cli/session_test.go`

**File**: `internal/cli/session_test.go`

Table-driven tests covering:
- `NewSession` success for Ed25519, ECDSA (P-256), and RSA (2048-bit) keys.
- `NewSession` failure for an unsupported key type → non-nil error.
- `Decrypt` round-trip: encrypt with `internal/encryption.Encrypt` directly, decrypt via `session.Decrypt`.
- `Decrypt` on a zero-value `Session` (nil `encKey`) → `ErrNotAuthenticated`.
- `Decrypt` with a tampered/invalid ciphertext → wraps `ErrDecrypt`.

Re-use key generation helpers from `internal/encryption/encryption_test.go` or generate keys inline.

---

### 3. Document lazy session creation in `root.go`

**File**: `internal/cli/root.go`

- In `PersistentPreRunE`, add a comment block explaining:
  - `LoadIdentity` is called here to resolve the SSH identity.
  - Encryption session (`NewSession`) is **not** initialised globally; each command
    that needs decryption must call `NewSession(signer)` after loading the identity.
  - Reference `cli_export` and `cli_exec` as the first consumers.
- No functional code changes to `root.go` are required in this task.

---

### 4. Verify no regression

- Confirm `export.go` and `exec.go` stubs compile unchanged after adding `session.go`.
- Run `go build ./...` and `go test ./...` — all must pass.

