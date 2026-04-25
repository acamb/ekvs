# validation.md — tui_encryption

## Acceptance Criteria

The milestone is complete when **all** of the following conditions are met.

---

## 1. Unit tests pass

```bash
make test
# or directly:
go test ./internal/tui/session/...
```

- All table-driven test cases in `internal/tui/session/session_test.go` pass.
- Statement coverage for `internal/tui/session` is ≥ 90 %:

```bash
go test -coverprofile=coverage.out ./internal/tui/session/...
go tool cover -func=coverage.out | grep 'internal/tui/session'
```

---

## 2. Build is clean

```bash
go build ./...
```

No compilation errors, no unused imports.

---

## 3. Key derivation correctness

Verify that `SetAuthenticated` on the same key always produces the same `encKey`
(deterministic derivation):

```go
s1 := &session.Session{}
s1.SetAuthenticated(signer, pub, fp)

s2 := &session.Session{}
s2.SetAuthenticated(signer, pub, fp)

// s1.Encrypt(x) and s2.Decrypt(s1.Encrypt(x)) must round-trip correctly.
```

Covered by the table-driven tests in task 4.

---

## 4. Encrypt / Decrypt round-trip

For each supported key type (Ed25519, ECDSA P-256, RSA 2048):

1. Call `session.SetAuthenticated` with the test key.
2. Call `session.Encrypt("hello, world")`.
3. Verify the returned string is non-empty and different from the plaintext.
4. Call `session.Decrypt(<ciphertext>)`.
5. Verify the result equals `"hello, world"`.

Covered by unit tests; also verifiable manually with a small Go test program.

---

## 5. Error paths

| Scenario | Expected error |
|----------|----------------|
| `Encrypt` on unauthenticated session | `session.ErrNotAuthenticated` |
| `Decrypt` on unauthenticated session | `session.ErrNotAuthenticated` |
| `Decrypt` with tampered ciphertext | wraps `session.ErrDecrypt` |
| `SetAuthenticated` with unsupported key | non-nil error; session remains unauthenticated |

---

## 6. `Clear()` resets encryption state

After calling `session.Clear()`:
- `IsAuthenticated()` returns `false`.
- `Encrypt(…)` returns `ErrNotAuthenticated`.
- `Decrypt(…)` returns `ErrNotAuthenticated`.

---

## 7. Auth flow still works end-to-end

Launch the TUI against a running server:

```bash
go run ./cmd/tui/main.go --config ekvs-tui.yaml
```

1. Navigate to the Projects screen.
2. Confirm the auth passphrase prompt (if key is encrypted) or silent load.
3. Confirm the project list is displayed correctly.
4. No regression in the existing auth / projects flow.

