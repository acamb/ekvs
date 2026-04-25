# plan.md — tui_encryption

Ordered list of tasks to implement the `tui_encryption` feature.

---

## Tasks

### 1. Add error sentinels and `SetAuthenticated` to `session.Session`

**File**: `internal/tui/session/session.go`

- Add package-level error sentinels: `ErrNotAuthenticated`, `ErrEncrypt`, `ErrDecrypt`.
- Add unexported `encKey []byte` field to `Session`.
- Add `SetAuthenticated(signer crypto.Signer, pub gossh.PublicKey, fp string) error` method:
  - Call `encryption.DeriveKey(signer)`; return error on failure.
  - Store `signer`, `pub`, `fp`, and `encKey` on success.
- Update `Clear()` to zero `encKey` bytes before resetting the struct.

---

### 2. Add `Encrypt` and `Decrypt` methods to `session.Session`

**File**: `internal/tui/session/session.go` (continuation of task 1)

- `Encrypt(plaintext string) (string, error)`:
  - Return `ErrNotAuthenticated` if `encKey == nil`.
  - Delegate to `encryption.Encrypt(s.encKey, []byte(plaintext))`; wrap errors in `ErrEncrypt`.
- `Decrypt(encoded string) (string, error)`:
  - Return `ErrNotAuthenticated` if `encKey == nil`.
  - Delegate to `encryption.Decrypt(s.encKey, encoded)`; wrap errors in `ErrDecrypt`.

---

### 3. Update `internal/tui/auth` to use `SetAuthenticated`

**File**: `internal/tui/auth/auth.go` (or wherever session fields are set after successful key loading)

- Replace direct field assignments (`s.Signer = …`, `s.Fingerprint = …`) with a call to `session.SetAuthenticated(signer, pub, fp)`.
- On error, emit an `auth.AuthErrMsg` (same path as existing auth errors) so the UI can display the failure.

---

### 4. Write unit tests for `session.Session`

**File**: `internal/tui/session/session_test.go`

Table-driven tests covering:
- `SetAuthenticated` success for Ed25519, ECDSA (P-256), and RSA (2048-bit) keys.
- `SetAuthenticated` failure for an unsupported key type.
- `Encrypt` / `Decrypt` round-trip for each supported key type.
- `Encrypt` and `Decrypt` on an unauthenticated session → `ErrNotAuthenticated`.
- `Decrypt` with a tampered ciphertext → `ErrDecrypt`.
- `Clear()` resets all fields; subsequent `Encrypt` returns `ErrNotAuthenticated`.

Reuse PEM fixtures from `internal/ssh/testdata/`.

---

### 5. Verify no other changes required

- Confirm `internal/tui/client` does not need changes (it uses `session.Signer` / `session.Fingerprint` only).
- Confirm `internal/tui/projects` does not need changes (it does not deal with encryption).
- Run `make test` and ensure all existing tests pass.

