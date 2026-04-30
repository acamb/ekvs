# plan.md — cli_exec

Ordered task list. Each task must pass `make test` before the next begins.

---

## Task 1 — Implement `internal/cli/exec.go`

**File**: `internal/cli/exec.go` (replace stub)

Replace the stub `RunE` with a full implementation following the pattern of `export.go`.

Steps inside `RunE`:

1. `LoadIdentity(flagIdentity, flagPassphrase)` → `signer, _, fingerprint, err`.
2. `NewSession(signer)` → `sess, err`.
3. `NewClient("http://" + flagServer)`.
4. Split args using `cmd.ArgsLenAtDash()`:
   - `toolArgs := args[:cmd.ArgsLenAtDash()]`
   - `programArgs := args[cmd.ArgsLenAtDash():]`
   - `project = toolArgs[0]`; if `len(toolArgs) == 2`, `keyName = toolArgs[1]`.
   - `program = programArgs[0]`, `extraArgs = programArgs[1:]`.
5. Fetch and decrypt:
   - **Single key**: `client.GetSecret(signer, fingerprint, project, keyName)` → `sess.Decrypt(entry.Value)` → one `"KEY=value"` string.
   - **All keys**: `client.ListSecrets(signer, fingerprint, project)` → `sess.Decrypt` for each entry → slice of `"KEY=value"` strings. Wrap `ErrNotFound` with `"project not found"`.
6. Build env slice: `append(os.Environ(), decryptedPairs...)`.
7. Create subprocess: `exec.Command(program, extraArgs...)` with:
   - `Env` set to the env slice above.
   - `Stdin = os.Stdin`, `Stdout = os.Stdout`, `Stderr = os.Stderr`.
8. Return the result of `subCmd.Run()`.

---

## Task 2 — Write unit tests in `internal/cli/exec_test.go`

**File**: `internal/cli/exec_test.go` (new file, package `cli_test`)

Use the same test helpers as `export_test.go` (`loadTestEd25519`, encryption round-trip, `httptest.NewServer`, `ExecuteWithArgs`).

**Subprocess testing strategy**: use the *test-helper process pattern*. If
`os.Getenv("EKVS_EXEC_HELPER")` equals a sentinel string, the test binary
prints a specific env var to stdout and exits 0. Tests invoke `os.Executable()`
as the subprocess program and inject the sentinel via the secret env vars,
allowing assertion on injected values without relying on external binaries.

Table-driven tests to implement (see `validation.md` for the full list):

- Happy path: all secrets → subprocess receives all env entries.
- Happy path: single key → only that env entry is injected.
- Empty secrets list → subprocess runs, no extra env vars.
- 404 project → error `"project not found"`.
- 404 key → error `"project or key not found"`.
- Program not found → non-nil exec error.
- Program exits non-zero → `*exec.ExitError` returned.
- Extra program args forwarded correctly.
- Parent env is preserved in subprocess env.

---

## Task 3 — Final check

- `go test -race ./internal/cli/...` passes with no failures.
- `go build ./...` succeeds.
- Manual smoke-test per `validation.md §§ 3–5`.

