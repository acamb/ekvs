# plan.md — encryption_primitives

## Ordered Task List

1. **Create package skeleton**
   Create `internal/encryption/` with the following source files:
   - `errors.go` — package-level error sentinels
   - `derive.go` — `DeriveKey` implementation
   - `cipher.go` — `Encrypt` and `Decrypt` implementations
   - `encryption_test.go` — all unit tests

2. **Define error sentinels**
   In `internal/encryption/errors.go`, declare:
   - `ErrUnsupportedKeyType = errors.New("unsupported key type")`
   - `ErrDecryptionFailed   = errors.New("decryption failed")`

3. **Implement `DeriveKey` — Ed25519 branch**
   In `derive.go`, handle `ed25519.PrivateKey`: extract the first 32 bytes (the seed),
   feed into `hkdf.New(sha256.New, seed, nil, []byte("ekvs-encryption-key-v1"))`, read 32 bytes.

4. **Implement `DeriveKey` — ECDSA branch**
   Handle `*ecdsa.PrivateKey`: extract `key.D.Bytes()` as the IKM, pass through the same HKDF call.

5. **Implement `DeriveKey` — RSA branch**
   Handle `*rsa.PrivateKey`:
   - Reject keys with `key.N.BitLen() < 2048` → return `ErrUnsupportedKeyType`.
   - Concatenate **all** prime factors: `key.Primes[0].Bytes() ∥ … ∥ key.Primes[n-1].Bytes()`.
   - Pass the concatenated bytes through the same HKDF call.

6. **Implement `Encrypt`**
   In `cipher.go`: create AES-256-GCM cipher from the 32-byte key, generate a 12-byte random nonce
   via `crypto/rand`, seal the plaintext (producing ciphertext+tag), return
   `base64.StdEncoding.EncodeToString(nonce || ciphertext+tag)`.

7. **Implement `Decrypt`**
   In `cipher.go`: base64-decode the input, split the first 12 bytes as the nonce, call `gcm.Open`;
   on any failure return `ErrDecryptionFailed`.

8. **Write unit tests (table-driven) in `encryption_test.go`**
   Cover:
   - `DeriveKey` × {RSA-2048, ECDSA-P256, Ed25519} — output is 32 bytes, deterministic
   - `DeriveKey` with RSA key < 2048 bit → `ErrUnsupportedKeyType`
   - `DeriveKey` with an unsupported key type → `ErrUnsupportedKeyType`
   - `Encrypt`/`Decrypt` round-trip × all key types
   - `Encrypt` with same key + plaintext → different ciphertexts each call (nonce randomness)
   - `Decrypt` with tampered ciphertext, truncated blob, invalid base64, wrong key → `ErrDecryptionFailed`
   - Reuse PEM fixtures from `internal/ssh/testdata/` via `internal/ssh.ParsePrivateKey`

9. **Run `go mod tidy` and validate**
   Run `go mod tidy` (no new deps — `golang.org/x/crypto/hkdf` already in `go.mod`),
   then `make test`; confirm `ok  ekvs/internal/encryption`.
