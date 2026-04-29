# validation.md — cli_auth

## Automated checks

```
go build ./...
go test ./...
```

Must pass after every individual task in `plan.md`.

---

## Checklist

### `--passphrase` flag (Task 1)

- [ ] `ekvs --help` shows `--passphrase` in the flags list.
- [ ] `EKVS_PASSPHRASE=secret ekvs export ...` resolves passphrase from env
  (verified via unit test).
- [ ] `--passphrase` flag overrides env (verified via unit test in `cli_test.go`).

### `LoadIdentity` (Task 3)

- [ ] Unencrypted ed25519 test key loads without error.
- [ ] Returned fingerprint matches `ssh.FingerprintSHA256` of the same key
  computed independently.
- [ ] Non-existent path returns a non-nil error containing the path.
- [ ] (Manual) Encrypted key with correct passphrase via `--passphrase` loads
  successfully.
- [ ] (Manual) Encrypted key without passphrase prompts on stderr and reads
  from stdin.

### `SignedHeaders` (Task 4)

- [ ] Returned map has exactly three keys: `X-Timestamp`, `X-Fingerprint`,
  `X-Signature`.
- [ ] `X-Timestamp` is a valid Unix epoch integer string.
- [ ] `X-Signature` is valid base64.
- [ ] Round-trip: decode `X-Signature`, unmarshal as `gossh.Signature`,
  verify against the canonical request using `internal/ssh.Verify` → no error.

### Unit tests (Task 5)

- [ ] `go test ./internal/cli/... -v` shows all four test cases passing.
- [ ] No race conditions: `go test -race ./internal/cli/...` passes.

