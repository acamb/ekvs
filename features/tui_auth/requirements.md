# requirements.md — tui_auth

## Goal

Implement SSH key loading and request-signing capabilities in the TUI client.
Authentication happens lazily: the private key is loaded (and the passphrase
prompt shown, if needed) only the first time an operation that requires an
authenticated HTTP request is triggered.  The loaded key is kept in the session
state for the entire TUI session and is never written back to disk.

---

## Scope

### In scope
- `internal/tui/session` package: holds in-memory session state (signer,
  fingerprint, passphrase).
- `internal/tui/auth` package: bubbletea model for the passphrase input screen.
- `internal/ssh` extension: `ParsePrivateKeyWithPassphrase` helper for
  passphrase-protected keys, and `ErrPassphraseRequired` sentinel.
- Integration of the auth flow into the root model: before the first
  authenticated API call the auth model is shown; on success control returns to
  the caller.
- A `SignRequest` helper that, given a session, builds the three HTTP headers
  (`X-Timestamp`, `X-Fingerprint`, `X-Signature`) expected by the server auth
  middleware (`internal/auth.AuthMiddleware`).
- Unit tests for all new packages.

### Out of scope
- SSH agent support (planned for **ssh_agent_support**, Phase 6).
- Any actual HTTP calls to the server (deferred to `tui_projects` /
  `tui_secrets`).
- CLI client (separate milestone).

---

## Decisions

| # | Decision |
|---|----------|
| 1 | The `identity_file` from the active profile is the sole source of the private key; no interactive file picker is shown. |
| 2 | The passphrase prompt is shown only when the key file is passphrase-protected. Unencrypted keys are loaded silently. |
| 3 | The passphrase prompt is shown **once per session**, the first time an authenticated operation is triggered. |
| 4 | Session state (signer, fingerprint) lives in a `session.Session` struct stored in the root model; it is cleared when the profile changes (enforced in `tui_profiles`). |
| 5 | SSH agent support is explicitly deferred to Phase 6. |
| 6 | `internal/ssh.ParsePrivateKeyWithPassphrase` wraps `golang.org/x/crypto/ssh.ParseRawPrivateKeyWithPassphrase`. |
| 7 | `ErrPassphraseRequired` is added to `internal/ssh/errors.go`. `ParsePrivateKey` returns it when `golang.org/x/crypto/ssh.ParseRawPrivateKey` fails and the error is of type `*gossh.PassphraseMissingError`. |
| 8 | The `crypto.Signer` stored in `session.Session` is the same type returned by `internal/ssh.ParsePrivateKey` / `ParsePrivateKeyWithPassphrase` and is directly usable by `internal/ssh.Sign` and `internal/encryption.DeriveKey`. |

---

## Session state

```go
// internal/tui/session/session.go
package session

import (
    "crypto"
    gossh "golang.org/x/crypto/ssh"
)

// Session holds per-session authentication state.
// The zero value represents an unauthenticated session.
// Signer is kept as crypto.Signer so it can be passed directly to:
//   - internal/ssh.Sign         (request signing)
//   - internal/encryption.DeriveKey (symmetric key derivation — tui_encryption)
type Session struct {
    Signer      crypto.Signer   // nil until authenticated
    PublicKey   gossh.PublicKey // nil until authenticated
    Fingerprint string          // empty until authenticated
}

func (s *Session) IsAuthenticated() bool { return s.Signer != nil }

// Clear resets all authentication state (called on profile switch).
func (s *Session) Clear() { *s = Session{} }
```

---

## Auth flow (lazy)

```
User triggers an operation requiring auth
        │
        ▼
session.IsAuthenticated()?
  YES ──► proceed to API call (future milestones)
  NO
        │
        ▼
ssh.ParsePrivateKey(identityFile)
  success (unencrypted key) ──► store in session ──► proceed
  ErrPassphraseRequired
        │
        ▼
Show passphrase input screen (internal/tui/auth.Model)
        │
  user submits passphrase
        │
        ▼
ssh.ParsePrivateKeyWithPassphrase(identityFile, passphrase)
  success ──► store in session ──► return to previous screen
  failure ──► show error message, allow retry
```

---

## HTTP header signing

`internal/tui/auth.SignRequest(sess *session.Session, method, path string, now time.Time) (map[string]string, error)`

Builds the three headers expected by the server `AuthMiddleware`:

| Header | Value | Notes |
|--------|-------|-------|
| `X-Timestamp` | `strconv.FormatInt(now.UTC().Unix(), 10)` | Must match the timestamp embedded in the signed message |
| `X-Fingerprint` | `sess.Fingerprint` | Canonical `SHA256:<base64>` string from `ssh.FingerprintSHA256` |
| `X-Signature` | `base64.StdEncoding.EncodeToString(sigBlob)` | `sigBlob` = `ssh.Marshal(sig)` produced by `ssh.Sign` |

The signed message is `ssh.CanonicalRequest(method, path, now)`, which produces:

```
{METHOD}\n{PATH}\n{unix_timestamp_seconds}
```

**Critical**: the same `now` value must be used for both `X-Timestamp` and the
canonical request string, so that the server can reconstruct the exact message
that was signed (`internalssh.CanonicalRequest(r.Method, r.URL.Path, tsTime)`
where `tsTime = time.Unix(tsUnix, 0)` and `tsUnix` is parsed from `X-Timestamp`).

Returns `ErrNotAuthenticated` if `sess.Signer == nil`.

---

## Package layout

```
internal/
  ssh/
    keys.go                  (existing — add ParsePrivateKeyWithPassphrase)
  tui/
    session/
      session.go
      session_test.go
    auth/
      auth.go                (bubbletea model: passphrase input screen)
      auth_test.go
      sign.go                (SignRequest helper)
      sign_test.go
```

---

## Constraints

- No new external dependencies: use `golang.org/x/crypto/ssh` (already in
  `go.mod`) and the standard library.
- bubbletea v2 (`charm.land/bubbletea/v2`) for the passphrase input model.
- Table-driven unit tests, Go standard `testing` package.
- Statement coverage ≥ 90 % on new packages.





