# requirements.md — tui_encryption

## Goal

Wire the `internal/encryption` package into the TUI session context so that the
upcoming `tui_secrets` screen can encrypt and decrypt secret values without
knowing the encryption internals.  No UI changes are required — this milestone
is purely a plumbing/wiring task.

---

## Scope

### In scope
- Extend `internal/tui/session/session.go`:
  - Add unexported `encKey []byte` field to `Session`.
  - Add `SetAuthenticated(signer crypto.Signer, pub gossh.PublicKey, fp string) error` constructor that sets `Signer`, `PublicKey`, `Fingerprint`, **and** derives+caches the AES key via `encryption.DeriveKey`.
  - Add `Encrypt(plaintext string) (string, error)` method.
  - Add `Decrypt(encoded string) (string, error)` method.
  - Update `Clear()` to zero `encKey` bytes before nil-ing the field.
  - Add session-level error sentinels: `ErrNotAuthenticated`, `ErrEncrypt`, `ErrDecrypt`.
- Update `internal/tui/auth` (wherever it currently populates the session) to
  call `session.SetAuthenticated(…)` instead of setting fields directly.
- Unit tests for all new session behaviour (table-driven).

### Out of scope
- Any UI or bubbletea model changes.
- Secret management screen (deferred to `tui_secrets`).
- Key rotation or re-encryption of stored values.
- SSH agent support (deferred to `ssh_agent_support`, Phase 6).
- CLI client (separate milestones).

---

## Decisions

| # | Decision |
|---|----------|
| 1 | Key derivation is **eager**: it happens in `SetAuthenticated`, not lazily on first `Encrypt`/`Decrypt` call. This keeps `Encrypt`/`Decrypt` error-free with respect to derivation and avoids hidden lazy-init complexity. |
| 2 | `SetAuthenticated` replaces direct field assignment in `tui_auth`. The `tui_auth` package is updated to call `session.SetAuthenticated` so the session is always in a consistent state after authentication. |
| 3 | `Encrypt` and `Decrypt` wrap raw `internal/encryption` errors in session-level sentinels (`ErrEncrypt`, `ErrDecrypt`) so that `tui_secrets` consumers do not need to import `internal/encryption`. |
| 4 | `Clear()` zeros `encKey` bytes in memory (`for i := range s.encKey { s.encKey[i] = 0 }`) before nil-ing the slice, to minimise the window during which key material is present in the heap. |
| 5 | `ErrNotAuthenticated` is returned by `Encrypt` and `Decrypt` when `encKey` is nil (session not yet authenticated). |
| 6 | `encKey` is unexported; consumers interact only through `Encrypt`/`Decrypt`. |

---

## Updated `session.Session`

```go
// internal/tui/session/session.go
package session

import (
    "crypto"
    "errors"
    "fmt"

    "ekvs/internal/encryption"
    gossh "golang.org/x/crypto/ssh"
)

var (
    // ErrNotAuthenticated is returned when Encrypt or Decrypt is called on an
    // unauthenticated session.
    ErrNotAuthenticated = errors.New("session: not authenticated")

    // ErrEncrypt wraps encryption errors from internal/encryption.
    ErrEncrypt = errors.New("session: encrypt failed")

    // ErrDecrypt wraps decryption errors from internal/encryption.
    ErrDecrypt = errors.New("session: decrypt failed")
)

// Session holds per-session authentication and encryption state.
// The zero value represents an unauthenticated session.
type Session struct {
    Signer      crypto.Signer   // nil until authenticated
    PublicKey   gossh.PublicKey // nil until authenticated
    Fingerprint string          // empty until authenticated
    encKey      []byte          // 32-byte AES-256 key; nil until authenticated
}

// SetAuthenticated authenticates the session and derives the symmetric
// encryption key from signer.  Returns an error if key derivation fails.
func (s *Session) SetAuthenticated(signer crypto.Signer, pub gossh.PublicKey, fp string) error {
    key, err := encryption.DeriveKey(signer)
    if err != nil {
        return fmt.Errorf("session: derive key: %w", err)
    }
    s.Signer = signer
    s.PublicKey = pub
    s.Fingerprint = fp
    s.encKey = key
    return nil
}

// IsAuthenticated returns true if the session has been authenticated.
func (s *Session) IsAuthenticated() bool { return s.Signer != nil }

// Encrypt encrypts plaintext using the session's derived AES-256-GCM key.
// Returns ErrNotAuthenticated if the session is not authenticated.
// Encryption errors are wrapped in ErrEncrypt.
func (s *Session) Encrypt(plaintext string) (string, error) {
    if s.encKey == nil {
        return "", ErrNotAuthenticated
    }
    ct, err := encryption.Encrypt(s.encKey, []byte(plaintext))
    if err != nil {
        return "", fmt.Errorf("%w: %w", ErrEncrypt, err)
    }
    return ct, nil
}

// Decrypt decrypts a base64-encoded ciphertext blob produced by Encrypt.
// Returns ErrNotAuthenticated if the session is not authenticated.
// Decryption errors are wrapped in ErrDecrypt.
func (s *Session) Decrypt(encoded string) (string, error) {
    if s.encKey == nil {
        return "", ErrNotAuthenticated
    }
    pt, err := encryption.Decrypt(s.encKey, encoded)
    if err != nil {
        return "", fmt.Errorf("%w: %w", ErrDecrypt, err)
    }
    return string(pt), nil
}

// Clear resets all authentication and encryption state.
// encKey bytes are zeroed before the slice is nil-ed to reduce the window
// during which key material is present in the heap.
// Called on profile switch.
func (s *Session) Clear() {
    for i := range s.encKey {
        s.encKey[i] = 0
    }
    *s = Session{}
}
```

---

## Changes to `internal/tui/auth`

Wherever `tui_auth` currently populates session fields directly (e.g.
`s.Signer = signer; s.Fingerprint = fp`), replace with a call to
`s.SetAuthenticated(signer, pub, fp)` and propagate any error as an
`auth.AuthErrMsg`.

---

## Testing requirements

- Table-driven unit tests in `internal/tui/session/session_test.go`.
- Cover:
  - `SetAuthenticated` succeeds with valid signer → `IsAuthenticated()` true, `encKey` non-nil.
  - `SetAuthenticated` with unsupported key type → returns error, session remains unauthenticated.
  - `Encrypt` / `Decrypt` round-trip for each supported key type (Ed25519, ECDSA, RSA).
  - `Encrypt` / `Decrypt` on unauthenticated session → `ErrNotAuthenticated`.
  - `Decrypt` with tampered ciphertext → `ErrDecrypt`.
  - `Clear()` resets all fields; subsequent `Encrypt` returns `ErrNotAuthenticated`.
- Reuse PEM fixtures from `internal/ssh/testdata/`.
- Target ≥ 90 % statement coverage for `internal/tui/session`.

