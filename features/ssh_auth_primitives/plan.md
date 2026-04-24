# plan.md — ssh_auth_primitives

## Ordered Task List

1. **Add dependency**
   Run `go get golang.org/x/crypto/ssh` and commit the updated `go.mod` / `go.sum`.

2. **Create package skeleton**
   Create `internal/ssh/` with the following source files:
   - `errors.go` — package-level error sentinels
   - `keys.go` — parsing and fingerprinting
   - `sign.go` — canonical request string + client-side signing
   - `verify.go` — server-side verification + replay protection

3. **Define package-level error sentinels**
   In `internal/ssh/errors.go`, declare:
   - `ErrInvalidSignature`
   - `ErrKeyNotFound`
   - `ErrReplayDetected`
   - `ErrUnsupportedKeyType`

4. **Implement key parsing (`keys.go`)**
   - `ParsePrivateKey(pemBytes []byte) (crypto.Signer, ssh.PublicKey, error)` — decode PEM block, call `ssh.ParseRawPrivateKey`, type-assert to `crypto.Signer`, derive the `ssh.PublicKey` via `ssh.NewSignerFromKey`.
   - `ParseAuthorizedKey(line []byte) (ssh.PublicKey, error)` — thin wrapper around `ssh.ParseAuthorizedKey`; strip the trailing comment.

5. **Implement fingerprinting (`keys.go`)**
   - `Fingerprint(pub ssh.PublicKey) string` — compute `ssh.FingerprintSHA256(pub)`; return the `SHA256:…` string.

6. **Implement canonical request builder + signing (`sign.go`)**
   - `CanonicalRequest(method, path string, ts time.Time) string` — returns `"{METHOD}\n{PATH}\n{unix_seconds}"`.
   - `Sign(signer crypto.Signer, message []byte) ([]byte, error)` — create an `ssh.Signer` via `ssh.NewSignerFromSigner`, call `.Sign(rand.Reader, message)`, serialise the `ssh.Signature` struct via `ssh.Marshal` and return the resulting bytes.

7. **Implement verification + replay protection (`verify.go`)**
   - `Verify(pub ssh.PublicKey, message, sigBlob []byte) error` — deserialise the blob via `ssh.Unmarshal` into an `ssh.Signature`, call `pub.Verify(message, sig)`; return `ErrInvalidSignature` on failure.
   - `CheckTimestamp(ts time.Time, window time.Duration) error` — compare `ts` to `time.Now().UTC()`; return `ErrReplayDetected` if `|delta| > window`. Caller passes `30 * time.Second` as the standard window.

8. **Generate test fixtures**
   Add `internal/ssh/testdata/` with pre-generated unencrypted PEM keys and their `authorized_keys` lines (one set per supported key type):
   ```zsh
   cd internal/ssh/testdata
   ssh-keygen -t ed25519     -N "" -f ed25519   -C "test-ed25519"
   ssh-keygen -t ecdsa       -N "" -f ecdsa     -C "test-ecdsa"
   ssh-keygen -t rsa -b 2048 -N "" -f rsa       -C "test-rsa"
   ```

9. **Write unit tests (`ssh_test.go`)**
   Table-driven tests covering:
   - `ParsePrivateKey` × {RSA-2048, ECDSA-P256, Ed25519}
   - `ParseAuthorizedKey` × same key types + malformed input
   - `Fingerprint` — compare against known `ssh-keygen -lf` output
   - `CanonicalRequest` — format correctness
   - `Sign` + `Verify` round-trip × {RSA, ECDSA, Ed25519}
   - `Verify` with tampered message / signature / wrong key → `ErrInvalidSignature`
   - `CheckTimestamp` — within window / outside window / exactly on boundary

10. **Run `go mod tidy` and validate**
    Run `go mod tidy`, then `make test`; confirm `ok  ekvs/internal/ssh`.

