# validation.md — cli_encryption

## Automated checks

```
go build ./...
go test ./...
```

Must pass after every individual task in `plan.md`.

---

## Checklist

### `NewSession` (Task 1)

| Case | Expected |
|------|----------|
| Ed25519 key | `*Session` returned, no error |
| ECDSA P-256 key | `*Session` returned, no error |
| RSA 2048 key | `*Session` returned, no error |
| Unsupported key type | `nil`, non-nil error |

### `Decrypt` (Task 1)

| Case | Expected |
|------|----------|
| Valid round-trip (encrypt with `encryption.Encrypt`, decrypt via `session.Decrypt`) | decrypted plaintext matches original |
| Zero-value `Session` (nil `encKey`) | `ErrNotAuthenticated` |
| Tampered base64 blob | error wrapping `ErrDecrypt` |

### Unit tests (Task 2)

- [ ] `go test -race ./internal/cli/...` passes with no failures.
- [ ] All pre-existing tests in `auth_test.go` and `cli_test.go` continue to pass.

### No regression (Task 4)

- [ ] `export.go` and `exec.go` stubs compile unchanged.
- [ ] `go build ./...` produces no errors or unused-import warnings.

