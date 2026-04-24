# requirements.md — encryption_primitives

## User Decisions

| Decision | Choice |
|---|---|
| Key derivation algorithm | HKDF-SHA256; info string `"ekvs-encryption-key-v1"`; empty salt |
| Symmetric cipher | AES-256-GCM (32-byte key, 12-byte nonce, 16-byte tag) |
| Nonce strategy | Fresh `crypto/rand` nonce per `Encrypt` call |
| Ciphertext encoding | `base64(nonce ∥ ciphertext+tag)` — standard (padded) base64 |
| Supported key types | RSA (≥ 2048 bit only), ECDSA (P-256, P-384, P-521), Ed25519 |
| Key derivation approach | Approach A — derive from raw private key material via HKDF (not hybrid/asymmetric wrapping) |
| Package name | `encryption` (path `internal/encryption`) — avoids stylistic conflict with stdlib `crypto` |
| RSA IKM | Concatenation of **all** primes: `key.Primes[0] ∥ key.Primes[1] ∥ … ∥ key.Primes[n-1]` |
| RSA minimum key size | Keys with modulus `< 2048 bit` are rejected with `ErrUnsupportedKeyType` |

---

## Scope

### In scope
- `DeriveKey(crypto.Signer) ([]byte, error)` — deterministic 32-byte AES key from SSH private key.
- `Encrypt(key, plaintext []byte) (string, error)` — AES-256-GCM encryption, base64 output.
- `Decrypt(key []byte, encoded string) ([]byte, error)` — inverse of `Encrypt`.
- Package-level error sentinels: `ErrUnsupportedKeyType`, `ErrDecryptionFailed`.
- Unit tests (table-driven) with ≥ 90 % statement coverage.
- Reuse of existing PEM fixtures from `internal/ssh/testdata/`.

### Out of scope
- Key caching or memoisation.
- Key rotation or re-encryption of stored values.
- Passphrase-protected private keys (deferred to backlog).
- Authenticated Additional Data (AAD) in GCM.
- Any network or storage I/O.
- Compression of plaintext.

---

## Package Path

```
internal/encryption
```

Import path: `ekvs/internal/encryption`

---

## Exported API

```go
package encryption

import "crypto"

var (
    ErrUnsupportedKeyType = errors.New("unsupported key type")
    ErrDecryptionFailed   = errors.New("decryption failed")
)

// DeriveKey derives a 32-byte AES-256 key from the given crypto.Signer.
// Supports *rsa.PrivateKey (≥ 2048 bit), *ecdsa.PrivateKey, ed25519.PrivateKey.
// Returns ErrUnsupportedKeyType for unrecognised key types or RSA keys < 2048 bit.
func DeriveKey(signer crypto.Signer) ([]byte, error)

// Encrypt encrypts plaintext using AES-256-GCM with the provided 32-byte key.
// Returns a base64-encoded string of the form base64(nonce || ciphertext+tag).
func Encrypt(key, plaintext []byte) (string, error)

// Decrypt decrypts a base64-encoded ciphertext blob produced by Encrypt.
// Returns the plaintext bytes or ErrDecryptionFailed on any failure.
func Decrypt(key []byte, encoded string) ([]byte, error)
```

---

## Key Derivation Specification

| Key type | HKDF Input Key Material (IKM) |
|---|---|
| `ed25519.PrivateKey` | `privateKey[:32]` — the 32-byte seed (first half of the 64-byte key) |
| `*ecdsa.PrivateKey` | `key.D.Bytes()` — the private scalar in big-endian bytes |
| `*rsa.PrivateKey` | `key.Primes[0].Bytes() ∥ … ∥ key.Primes[n-1].Bytes()` — all prime factors concatenated in order |

RSA pre-condition: `key.N.BitLen() < 2048` → return `ErrUnsupportedKeyType` before any derivation.

HKDF parameters (all types):
- Hash: `crypto/sha256`
- Salt: `nil` (zero-length)
- Info: `[]byte("ekvs-encryption-key-v1")`
- Output length: **32 bytes**

---

## Ciphertext Format Specification

```
base64StdEncoding( NONCE || CIPHERTEXT+TAG )
```

| Field | Size |
|---|---|
| NONCE | 12 bytes (GCM standard nonce, randomly generated per call) |
| CIPHERTEXT | `len(plaintext)` bytes |
| TAG | 16 bytes (GCM authentication tag, appended by `gcm.Seal`) |

Encoding: `base64.StdEncoding` (padded, standard alphabet). The result is ASCII-safe and
suitable for storage as a string value in the key-value store.

---

## Error Sentinels

| Sentinel | Returned by | Condition |
|---|---|---|
| `ErrUnsupportedKeyType` | `DeriveKey` | The underlying type of `crypto.Signer` is not RSA, ECDSA, or Ed25519; or RSA key has modulus < 2048 bit |
| `ErrDecryptionFailed` | `Decrypt` | Base64 decode error, blob < 12 bytes, or GCM authentication failure |

---

## Dependencies

No new module dependencies required. Packages used:

| Package | Use |
|---|---|
| `crypto/aes` | AES block cipher |
| `crypto/cipher` | GCM AEAD wrapper |
| `crypto/rand` | Random nonce generation |
| `crypto/sha256` | HKDF hash function |
| `encoding/base64` | Ciphertext encoding |
| `golang.org/x/crypto/hkdf` | Key derivation (already in `go.mod`) |

---

## Testing Requirements

- Framework: standard `testing` package; table-driven tests.
- Fixtures: PEM files in `internal/ssh/testdata/` (`rsa`, `ecdsa`, `ed25519`) loaded via
  `os.ReadFile` + `ekvs/internal/ssh.ParsePrivateKey`.
- Coverage target: **≥ 90 % statement coverage** for `ekvs/internal/encryption`.
- No external test helpers or test-only dependencies.

