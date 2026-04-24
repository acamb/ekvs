# validation.md — ssh_auth_primitives

## Manual Checklist

- [ ] `go.mod` lists `golang.org/x/crypto` as a direct dependency; `go.sum` is up to date.
- [ ] `internal/ssh/` contains: `errors.go`, `keys.go`, `sign.go`, `verify.go`, `ssh_test.go`.
- [ ] `internal/ssh/testdata/` contains pre-generated PEM files:
  - `ed25519`, `ed25519.pub`
  - `ecdsa`, `ecdsa.pub`
  - `rsa`, `rsa.pub`
- [ ] `ParsePrivateKey` returns a non-nil `crypto.Signer` and `ssh.PublicKey` for all three key types.
- [ ] `Fingerprint` output for `testdata/ed25519` matches the output of `ssh-keygen -lf testdata/ed25519`.
- [ ] `Sign` + `Verify` round-trip succeeds for all three key types.
- [ ] `Verify` returns `ErrInvalidSignature` when the message is altered after signing.
- [ ] `CheckTimestamp` returns `nil` for `delta = 0 s`, `±29 s`; returns `ErrReplayDetected` for `delta = ±31 s`.
- [ ] `go vet ./internal/ssh/...` reports no warnings.

---

## Running Unit Tests

```zsh
go test -race -count=1 -cover ./internal/ssh/...
```

**"Passing" means:**
- Exit code `0`.
- Output line: `ok  ekvs/internal/ssh`
- No data races reported.
- Statement coverage ≥ 90 % (printed as `coverage: XX.X% of statements`).

---

## Key Test-Case Matrix

| Test case                       | Ed25519 | ECDSA-P256 | RSA-2048 | Notes |
|---------------------------------|---------|------------|----------|-------|
| `ParsePrivateKey` happy path    | ✓       | ✓          | ✓        | |
| `ParsePrivateKey` unsupported type | —    | —          | —        | Must return `ErrUnsupportedKeyType` |
| `ParseAuthorizedKey` happy path | ✓       | ✓          | ✓        | |
| `ParseAuthorizedKey` malformed  | —       | —          | —        | Must return error |
| `Fingerprint`                   | ✓       | ✓          | ✓        | Compare against `ssh-keygen -lf` |
| `CanonicalRequest` format       | —       | —          | —        | Key-type-independent |
| `Sign` + `Verify` round-trip    | ✓       | ✓          | ✓        | |
| `Verify` tampered message       | ✓       | ✓          | ✓        | Must return `ErrInvalidSignature` |
| `Verify` tampered signature     | ✓       | ✓          | ✓        | Must return `ErrInvalidSignature` |
| `Verify` wrong public key       | ✓       | ✓          | ✓        | Must return `ErrInvalidSignature` |
| `CheckTimestamp` in window      | —       | —          | —        | `delta = 0 s`, `±29 s` → `nil` |
| `CheckTimestamp` on boundary    | —       | —          | —        | `delta = ±30 s` → `ErrReplayDetected` (strictly greater than window is rejected) |
| `CheckTimestamp` outside window | —       | —          | —        | `delta = ±31 s` → `ErrReplayDetected` |

---

## Generating / Refreshing Test Fixtures

```zsh
cd internal/ssh/testdata
ssh-keygen -t ed25519     -N "" -f ed25519   -C "test-ed25519"
ssh-keygen -t ecdsa       -N "" -f ecdsa     -C "test-ecdsa"
ssh-keygen -t rsa -b 2048 -N "" -f rsa       -C "test-rsa"
```

The `.pub` files are in `authorized_keys` format and are used directly as fixtures
for `ParseAuthorizedKey`.

