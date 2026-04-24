# requirements.md — ssh_auth_primitives

## User Decisions

| Decision | Choice |
|---|---|
| Authentication mechanism | Per-request signature (method A): client signs `METHOD\nPATH\nTIMESTAMP` with private SSH key; server verifies with registered public key. Stateless. |
| Passphrase-protected keys | **Not supported** in this milestone (deferred to backlog). |
| SSH agent support | **Not supported** in this milestone (deferred to backlog). |

---

## Scope

### In scope
- Parsing unencrypted SSH private keys (PEM) → `crypto.Signer` + `ssh.PublicKey`.
- Parsing SSH public keys from `authorized_keys` line format.
- SHA-256 fingerprinting of public keys.
- Building the canonical request string (shared format between client and server).
- Client-side signing of a message with a `crypto.Signer`.
- Server-side signature verification.
- Replay-protection timestamp window check.
- Package-level typed error sentinels.
- Unit tests (table-driven) for every exported function.
- Test fixtures (pre-generated PEM files) under `internal/ssh/testdata/`.

### Out of scope
- Passphrase-protected (encrypted) private keys.
- SSH agent (`ssh-agent` / `SSH_AUTH_SOCK`) integration.
- Session-based or token-based authentication.
- Key rotation or revocation logic.
- Network transport or HTTP middleware (those live in server/client packages).
- `authorized_keys` file I/O — this package only parses individual lines.

---

## Package Path

```
internal/ssh
```

Import path: `ekvs/internal/ssh`

---

## Exported API Surface

```go
package ssh

import (
    "crypto"
    "time"
    gossh "golang.org/x/crypto/ssh"
)

// -- Error sentinels --

var (
    ErrInvalidSignature   = errors.New("invalid signature")
    ErrKeyNotFound        = errors.New("key not found")
    ErrReplayDetected     = errors.New("request timestamp outside acceptable window")
    ErrUnsupportedKeyType = errors.New("unsupported key type")
)

// -- Key parsing --

// ParsePrivateKey parses an unencrypted PEM-encoded SSH private key.
// Returns a crypto.Signer (for signing) and the corresponding ssh.PublicKey.
func ParsePrivateKey(pemBytes []byte) (crypto.Signer, gossh.PublicKey, error)

// ParseAuthorizedKey parses a single line in authorized_keys format.
func ParseAuthorizedKey(line []byte) (gossh.PublicKey, error)

// -- Fingerprinting --

// Fingerprint returns the SHA-256 fingerprint of pub in "SHA256:<base64>" format,
// identical to the output of `ssh-keygen -lf`.
func Fingerprint(pub gossh.PublicKey) string

// -- Signing (client-side) --

// CanonicalRequest builds the message to be signed/verified.
// Format: "{METHOD}\n{PATH}\n{unix_timestamp_seconds}"
func CanonicalRequest(method, path string, ts time.Time) string

// Sign signs message with signer and returns a serialised signature blob
// (ssh.Marshal of an ssh.Signature struct).
func Sign(signer crypto.Signer, message []byte) ([]byte, error)

// -- Verification (server-side) --

// Verify verifies that sigBlob is a valid signature over message by pub.
// Returns ErrInvalidSignature on failure.
func Verify(pub gossh.PublicKey, message, sigBlob []byte) error

// -- Replay protection --

// CheckTimestamp returns ErrReplayDetected if ts is not within ±window of
// the current UTC time.
func CheckTimestamp(ts time.Time, window time.Duration) error
```

---

## Canonical Request String Format

```
{METHOD}\n{PATH}\n{UNIX_TIMESTAMP}
```

- `METHOD` — HTTP verb in uppercase, e.g. `GET`, `PUT`.
- `PATH` — request path including leading `/`, no query string, e.g. `/projects/myproject`.
- `UNIX_TIMESTAMP` — UTC Unix epoch **seconds** as a base-10 integer string.

Example:
```
GET
/projects/myproject
1745481600
```

---

## Signature Serialisation

`Sign` serialises the `ssh.Signature` struct using `ssh.Marshal`. `Verify` deserialises
with `ssh.Unmarshal` before calling `pub.Verify`. This keeps the encoding inside the
`golang.org/x/crypto/ssh` ecosystem without additional dependencies.

---

## Accepted Key Types

| Type    | Parameters          | Notes                         |
|---------|---------------------|-------------------------------|
| Ed25519 | —                   | Preferred; smallest keys      |
| ECDSA   | P-256 (min), P-384, P-521 also accepted | —          |
| RSA     | 2048-bit minimum    | Larger keys accepted          |

Any other type must return `ErrUnsupportedKeyType`.

---

## Error Sentinels

| Sentinel               | Returned by       | Meaning                                       |
|------------------------|-------------------|-----------------------------------------------|
| `ErrInvalidSignature`  | `Verify`          | Signature does not match public key/message   |
| `ErrKeyNotFound`       | (server layer)    | Reserved for key-lookup failures              |
| `ErrReplayDetected`    | `CheckTimestamp`  | Timestamp outside ±window                     |
| `ErrUnsupportedKeyType`| `ParsePrivateKey` | Key type decoded but not supported            |

---

## Testing Requirements

- All tests use the standard `testing` package and are table-driven.
- Test fixtures live in `internal/ssh/testdata/` and are committed to the repository.
- No network calls; no filesystem writes outside `testdata/`.
- Tests must pass with `go test -race -count=1 ./internal/ssh/...`.
- Statement coverage target: ≥ 90 %.

