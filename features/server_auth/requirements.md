# requirements.md — server_auth

## User Decisions

| Decision | Choice |
|----------|--------|
| Key management | **Filesystem-only**: the system administrator adds/removes public keys by placing `.pub` files in `{storage_root}/.keys/`. No HTTP API for key registration or revocation |
| Key file naming | Filenames are free-form (e.g. `alice_laptop.pub`, `bob_work.pub`) to aid human identification, but **must end in `.pub`**. The server ignores all other files in `.keys/` |
| Key file format | Each `.pub` file contains exactly one line in OpenSSH `authorized_keys` format: `{type} {base64} {comment}` |
| Key lookup strategy | On each `Lookup` call the server scans all `.pub` files in `.keys/`, parses each one, and matches by `FingerprintSHA256`. No in-memory cache (key management is infrequent; correctness > latency) |
| userID model | One SSH key = one userID. The fingerprint (`SHA256:<base64>`) IS the userID passed to `internal/storage`. No multi-key-per-account concept in this milestone |
| Signature headers | `X-Signature: <base64-encoded ssh.Marshal(sig)>`, `X-Timestamp: <unix-seconds>`, `X-Fingerprint: <SHA256:...>` |
| Timestamp window | **30 seconds** — as established in `ssh_auth_primitives` (`CheckTimestamp` caller contract). Passed as `window time.Duration` so tests can override it |
| Middleware context key | The verified fingerprint is stored in `context.Context` under an unexported key `ctxKeyUserID`; handlers retrieve it via `UserIDFromContext(ctx)` |
| Package path | `internal/auth` |
| Error responses | JSON body `{"error": "<message>"}` with appropriate HTTP status codes |

---

## Scope

### In scope
- `KeyStore` struct: **read-only** view of the `.keys/` directory (`Lookup`, `List`).
- `AuthMiddleware`: HTTP middleware that authenticates every request using SSH signatures; injects userID into context; rejects unknown keys and invalid/replayed signatures.
- Helper: `UserIDFromContext(ctx context.Context) (string, bool)`.
- Package-level error sentinels: `ErrKeyNotFound`, `ErrInvalidSignature`, `ErrReplayDetected`.
- Unit tests (table-driven) with ≥ 90% statement coverage.

### Out of scope
- Any HTTP endpoint for key management (register / list / revoke). Key lifecycle is managed by the system administrator via the filesystem.
- Project/secret HTTP endpoints (those live in `server_projects_api` / `server_secrets_api`).
- In-memory caching of key lookups or filesystem watches for hot-reload.
- Transport-level TLS.
- Rate limiting.

---

## Package Path

```
internal/auth
```

Import path: `ekvs/internal/auth`

---

## Key Storage Layout

```
{storage_root}/
  .keys/                        ← managed by the system administrator
    alice_laptop.pub            ← free-form filename, must end in .pub
    bob_work.pub
    carol_yubikey.pub
  {sanitized_fingerprint}/      ← per-user project directories (managed by storage package)
    project.json
```

Each `.pub` file contains exactly one line in OpenSSH `authorized_keys` format:

```
ssh-ed25519 AAAA...base64... alice@laptop
```

The comment field is free-form (typically `user@host` as produced by `ssh-keygen`). The server never writes to `.keys/`.

---

## Exported API

```go
package auth

import (
    "context"
    "errors"
    "net/http"
    "time"

    gossh "golang.org/x/crypto/ssh"
)

// Error sentinels.
var (
    ErrKeyNotFound      = errors.New("key not found")
    ErrInvalidSignature = errors.New("invalid signature")
    ErrReplayDetected   = errors.New("request timestamp outside acceptable window")
)

// KeyStore is a read-only view of the .keys/ directory.
// The system administrator manages keys by adding/removing .pub files.
type KeyStore struct { /* unexported */ }

// NewKeyStore opens a KeyStore rooted at keysDir.
// Returns an error if keysDir is not accessible.
func NewKeyStore(keysDir string) (*KeyStore, error)

// Lookup scans all .pub files in keysDir and returns the public key whose
// FingerprintSHA256 matches fingerprint.
// Returns ErrKeyNotFound if no match is found.
func (ks *KeyStore) Lookup(fingerprint string) (gossh.PublicKey, error)

// List returns the FingerprintSHA256 of every valid public key found in keysDir,
// sorted alphabetically. Files that cannot be parsed are silently skipped.
// Returns an empty non-nil slice if no valid keys are found.
func (ks *KeyStore) List() ([]string, error)

// AuthMiddleware returns an http.Handler that:
//   1. Reads X-Timestamp, X-Fingerprint and X-Signature headers.
//   2. Looks up the public key in KeyStore by fingerprint.
//   3. Reconstructs the canonical request string via internal/ssh.CanonicalRequest.
//   4. Decodes X-Signature from base64 and verifies it via internal/ssh.Verify.
//   5. Checks the timestamp is within window via internal/ssh.CheckTimestamp.
//   6. On success, injects the fingerprint as userID into the request context
//      and calls next.
//   7. On any failure, responds with 401 and a JSON error body.
//
// The standard window value is 30*time.Second, as established by ssh_auth_primitives.
func AuthMiddleware(ks *KeyStore, window time.Duration, next http.Handler) http.Handler

// UserIDFromContext retrieves the authenticated userID (fingerprint) from ctx.
// Returns ("", false) if the context was not populated by AuthMiddleware.
func UserIDFromContext(ctx context.Context) (string, bool)
```

---

## Authentication Flow

For every request to a protected endpoint:

| Header | Format | Example |
|--------|--------|---------|
| `X-Timestamp` | Unix seconds (decimal string) | `1745491200` |
| `X-Fingerprint` | Canonical SHA-256 fingerprint | `SHA256:abc+def/ghi=` |
| `X-Signature` | Base64-encoded `ssh.Marshal(sig)` blob | `AAAAB3Nza...` |

The signed message is:
```
{METHOD}\n{PATH}\n{unix_timestamp_seconds}
```
as produced by `internal/ssh.CanonicalRequest`.

`AuthMiddleware` verification steps:
1. Check all three headers are present → 401 if any is missing.
2. Parse `X-Timestamp` as int64 → 401 if malformed.
3. `ks.Lookup(X-Fingerprint)` → 401 if `ErrKeyNotFound`.
4. Reconstruct canonical message.
5. Base64-decode `X-Signature` → 401 if malformed.
6. `ssh.Verify(pub, message, sigBlob)` → 401 if `ErrInvalidSignature`.
7. `ssh.CheckTimestamp(ts, window)` → 401 if `ErrReplayDetected`.
8. `context.WithValue` with fingerprint → call `next`.

---

## Concurrency

`KeyStore` uses a single `sync.RWMutex` protecting `os.ReadDir` + file reads:
- Read lock for both `Lookup` and `List`.
- No write operations from the server side.

This prevents torn reads if the administrator is modifying `.keys/` while the server is running.

---

## Error Sentinels

| Sentinel | Returned by | Condition |
|----------|-------------|-----------|
| `ErrKeyNotFound` | `Lookup` | No `.pub` file matches the fingerprint |
| `ErrInvalidSignature` | `AuthMiddleware` | Signature does not verify; wraps `ssh.ErrInvalidSignature` |
| `ErrReplayDetected` | `AuthMiddleware` | Timestamp outside window; wraps `ssh.ErrReplayDetected` |

Callers should use `errors.Is` to inspect wrapped sentinels.

---

## Dependencies

No new module dependencies. Uses:
- Standard library: `context`, `encoding/base64`, `encoding/json`, `errors`, `fmt`, `net/http`, `os`, `path/filepath`, `sort`, `strconv`, `sync`, `time`.
- Internal packages: `ekvs/internal/ssh`.

---

## Testing Requirements

- Framework: standard `testing` package; table-driven tests.
- Each test uses `t.TempDir()` for full isolation; the test creates `.pub` files directly in the temp dir to simulate admin-managed keys.
- HTTP middleware tests use `net/http/httptest`.
- Test keys are loaded from `internal/ssh/testdata/` (existing RSA, ECDSA, Ed25519 key fixtures).
- Concurrency test: multiple goroutines calling `Lookup` and `List` simultaneously → no races (verified with `-race`).
- Coverage target: **≥ 90% statement coverage** for `ekvs/internal/auth`.


