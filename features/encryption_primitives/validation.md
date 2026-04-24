# validation.md — encryption_primitives

## How to Run Tests

```zsh
# Unit tests with race detector and coverage:
go test -race -count=1 -cover ./internal/encryption/...

# Detailed coverage report:
go test -count=1 -coverprofile=coverage.out ./internal/encryption/...
go tool cover -func=coverage.out

# Full suite (must not regress):
make test
```

**"Passing" definition:** all test cases exit with `PASS`, no race conditions, statement
coverage ≥ 90 % for `ekvs/internal/encryption`, and `make test` remains fully green.

---

## Manual Checklist

- [ ] `internal/encryption/errors.go` exports exactly `ErrUnsupportedKeyType` and `ErrDecryptionFailed`.
- [ ] `internal/encryption/derive.go` exports `DeriveKey` and handles all three key families.
- [ ] `internal/encryption/cipher.go` exports `Encrypt` and `Decrypt`.
- [ ] `go build ./internal/encryption/...` produces no errors.
- [ ] `go vet ./internal/encryption/...` produces no diagnostics.
- [ ] `go mod tidy` leaves `go.mod` and `go.sum` unchanged (no new deps needed).
- [ ] `Encrypt` called twice with the same key and plaintext produces **two different** base64 strings.
- [ ] `DeriveKey` called twice with the same in-memory signer returns **identical** byte slices.
- [ ] A ciphertext can be decoded by hand: base64-decode → first 12 bytes = nonce, rest = ciphertext+tag.
- [ ] `DeriveKey` with an RSA key < 2048 bit returns `ErrUnsupportedKeyType`.

---

## Test-Case Matrix

### `DeriveKey`

| # | Input | Expected |
|---|---|---|
| D-1 | `ed25519.PrivateKey` (from `testdata/ed25519`) | 32-byte slice, no error |
| D-2 | `*ecdsa.PrivateKey` P-256 (from `testdata/ecdsa`) | 32-byte slice, no error |
| D-3 | `*rsa.PrivateKey` 2048-bit (from `testdata/rsa`) | 32-byte slice, no error |
| D-4 | Same Ed25519 signer called twice | Both calls return identical bytes |
| D-5 | Same ECDSA signer called twice | Both calls return identical bytes |
| D-6 | Same RSA signer called twice | Both calls return identical bytes |
| D-7 | Unsupported signer type (unknown `Public()`) | `ErrUnsupportedKeyType` |
| D-8 | RSA key with modulus < 2048 bit | `ErrUnsupportedKeyType` |

### `Encrypt`

| # | Input | Expected |
|---|---|---|
| E-1 | Valid 32-byte key, non-empty plaintext | Non-empty base64 string, no error |
| E-2 | Valid 32-byte key, empty plaintext (`[]byte{}`) | Non-empty base64 string (nonce+tag only), no error |
| E-3 | Same key + plaintext called twice | Two different base64 strings (nonce randomness) |

### `Decrypt`

| # | Input | Expected |
|---|---|---|
| DC-1 | Output of `Encrypt` (Ed25519-derived key) | Plaintext equals original |
| DC-2 | Output of `Encrypt` (ECDSA-derived key) | Plaintext equals original |
| DC-3 | Output of `Encrypt` (RSA-derived key) | Plaintext equals original |
| DC-4 | Output of `Encrypt` with empty plaintext | Empty `[]byte`, no error |
| DC-5 | Valid blob but wrong decryption key | `ErrDecryptionFailed` |
| DC-6 | base64 of a blob shorter than 12 bytes | `ErrDecryptionFailed` |
| DC-7 | Valid base64, last byte of tag flipped | `ErrDecryptionFailed` |
| DC-8 | Invalid base64 string (e.g. `"not-base64!!"`) | `ErrDecryptionFailed` |

### Round-Trip

| # | Scenario |
|---|---|
| RT-1 | `DeriveKey(ed25519)` → `Encrypt` → `Decrypt` → plaintext equality |
| RT-2 | `DeriveKey(ecdsa)` → `Encrypt` → `Decrypt` → plaintext equality |
| RT-3 | `DeriveKey(rsa)` → `Encrypt` → `Decrypt` → plaintext equality |
| RT-4 | Encrypt three distinct values under the same key; decrypt all three independently |
