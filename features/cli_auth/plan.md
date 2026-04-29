# plan.md — cli_auth

Ordered list of atomic tasks. After each task `go build ./...` and
`go test ./...` must pass.

---

## Task 1 — Add `--passphrase` persistent flag to `rootCmd`

**Description**
In `internal/cli/root.go`:
- Declare `flagPassphrase string` package-level var.
- Register `--passphrase` as a persistent string flag (default `""`).
- In `PersistentPreRunE`, if `flagPassphrase == ""` check `EKVS_PASSPHRASE`
  env var and assign. Leave empty if not set (stdin prompt happens later
  inside `LoadIdentity`).

**Files**
- `internal/cli/root.go`

---

## Task 2 — Add `golang.org/x/term` dependency

**Description**
Run `go get golang.org/x/term` and ensure `go.mod` / `go.sum` are updated.
Verify `go build ./...` still passes.

**Files**
- `go.mod`, `go.sum`

---

## Task 3 — Implement `LoadIdentity` in `internal/cli/auth.go`

**Description**
Create `internal/cli/auth.go` with:

```go
// LoadIdentity loads an SSH private key from identityPath.
// passphrase is tried first; if empty and the key is encrypted, the function
// reads EKVS_PASSPHRASE from the environment, then falls back to prompting
// on stderr with echo disabled.
// Returns (signer, pubKey, fingerprint, error).
func LoadIdentity(identityPath, passphrase string) (crypto.Signer, gossh.PublicKey, string, error)
```

Implementation steps:
1. `os.ReadFile(identityPath)`.
2. `internalssh.ParsePrivateKey(pemBytes)` — if success, return.
3. If `errors.Is(err, internalssh.ErrPassphraseRequired)`:
   a. If `passphrase == ""` → check `os.Getenv("EKVS_PASSPHRASE")`.
   b. If still `""` → `fmt.Fprintf(os.Stderr, "Enter passphrase for %s: ", identityPath)` then `term.ReadPassword(int(os.Stdin.Fd()))`.
   c. Call `internalssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(passphrase))`.
4. Return `(signer, pub, internalssh.Fingerprint(pub), nil)`.

**Files**
- `internal/cli/auth.go`

---

## Task 4 — Implement `SignedHeaders` in `internal/cli/signer.go`

**Description**
Create `internal/cli/signer.go` with:

```go
// SignedHeaders produces the three HTTP authentication headers required by
// the EKVS server for the given method and path at time now.
// Returned keys: "X-Timestamp", "X-Fingerprint", "X-Signature".
func SignedHeaders(signer crypto.Signer, fingerprint, method, path string, now time.Time) (map[string]string, error)
```

Implementation:
1. `canonical := internalssh.CanonicalRequest(method, path, now)`.
2. `sigBlob, err := internalssh.Sign(signer, []byte(canonical))`.
3. Return `{"X-Timestamp": unix_str, "X-Fingerprint": fingerprint, "X-Signature": base64(sigBlob)}`.

**Files**
- `internal/cli/signer.go`

---

## Task 5 — Unit tests in `internal/cli/auth_test.go`

**Description**
Table-driven tests:
- `TestLoadIdentity_Unencrypted`: loads `internal/ssh/testdata/ed25519`, checks
  signer non-nil and fingerprint starts with `"SHA256:"`.
- `TestLoadIdentity_FileNotFound`: returns non-nil error.
- `TestSignedHeaders_Keys`: result map contains exactly
  `X-Timestamp`, `X-Fingerprint`, `X-Signature`.
- `TestSignedHeaders_RoundTrip`: signs a canonical request and verifies with
  `internal/ssh.Verify`.

**Files**
- `internal/cli/auth_test.go`

---

## Task 6 — Update existing CLI tests

**Description**
Update `internal/cli/cli_test.go` to account for the new `--passphrase` flag
(ensure existing flag-precedence tests still pass and add a case for
`EKVS_PASSPHRASE` env resolution).

**Files**
- `internal/cli/cli_test.go`

