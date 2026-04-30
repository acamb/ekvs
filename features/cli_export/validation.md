# validation.md — cli_export

## Automated checks

```
go build ./...
go test ./...
```

Must pass after every individual task in `plan.md`.

---

## Checklist

### `Client` (Task 1)

| Case | Expected |
|------|----------|
| `ListSecrets` — server returns `{"secrets":[…]}` | `[]secretEntry`, no error |
| `ListSecrets` — server returns 404 | `nil`, `ErrNotFound` |
| `GetSecret` — server returns `{"key":"k","value":"v"}` | `*secretEntry`, no error |
| `GetSecret` — server returns 404 | `nil`, `ErrNotFound` |
| Server returns non-200/non-404 status | `nil`, non-nil error (wraps status message) |

### `export` command (Task 2)

| Case | Expected |
|------|----------|
| `export proj` — multiple secrets | each `KEY=plaintext` line printed to stdout |
| `export proj key` — single secret | `KEY=plaintext\n` printed to stdout |
| `export proj` — empty project | no output, exit 0 |
| `export proj` — project not found (404) | non-zero exit, error message |
| `export proj key` — key not found (404) | non-zero exit, error message |
| Missing `--server` flag | error from `PersistentPreRunE` |
| Missing `--identity` flag | error from `PersistentPreRunE` |
| Encrypted blob round-trip (ed25519 key) | decrypted plaintext matches original |

### Unit tests (Task 3)

- [ ] `go test -race ./internal/cli/...` passes with no failures.
- [ ] All pre-existing tests in `auth_test.go`, `cli_test.go`, `session_test.go` continue to pass.

### No regression (Task 4)

- [ ] `exec.go` stub compiles unchanged.
- [ ] `go build ./...` produces no errors or unused-import warnings.

