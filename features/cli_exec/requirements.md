# requirements.md — cli_exec

## Goal

Implement `ekvs exec <projectName> [keyName] -- <program> [args...]`. The command
authenticates against the server, fetches secrets for the given project (one or
all), decrypts them, and injects them as environment variables into a subprocess.
The subprocess inherits the parent's stdin/stdout/stderr.

---

## Scope

### In scope

- Arg parsing using `cmd.ArgsLenAtDash()` to unambiguously separate the tool
  args (`projectName` + optional `keyName`) from the program and its args.
- Loading the SSH identity and deriving an encryption session (same pattern as
  `export.go`).
- Fetching secrets via `cli.Client`: `GetSecret` for a single key, `ListSecrets`
  for all.
- Decrypting each secret with `session.Decrypt`.
- Building the subprocess environment: `os.Environ()` + appended `KEY=value`
  pairs. Secret values take precedence over existing env vars with the same name
  (last-entry-wins semantics of `os/exec`).
- Spawning the subprocess with `os/exec`; inheriting `os.Stdin`, `os.Stdout`,
  `os.Stderr`.
- Propagating the subprocess exit error back to the caller.
- Unit tests in `internal/cli/exec_test.go` (table-driven, `httptest`,
  `ExecuteWithArgs`).

### Out of scope

- Filtering or removing existing environment variables from the parent.
- Signal forwarding to the subprocess.
- Secret-name transformation (key names are used verbatim as env var names).
- SSH agent support (separate milestone).
- Windows-specific process semantics.

---

## Decisions

| # | Decision |
|---|----------|
| 1 | **Arg disambiguation** — Use `cmd.ArgsLenAtDash()` to split `args` into *tool args* (before `--`) and *program args* (after `--`). If there is 1 tool arg it is the project name; if there are 2, the second is the key name. Program and its args occupy the remainder. This makes `ekvs exec proj -- cmd arg` vs `ekvs exec proj key -- cmd` unambiguous. |
| 2 | **Env injection** — Call `os.Environ()` to snapshot the parent environment, then append `KEY=value` for each decrypted secret. When a secret key collides with an existing env var, the secret value takes precedence (last entry wins for `os/exec`). |
| 3 | **Subprocess I/O** — Set `cmd.Stdin = os.Stdin`, `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr` so the child process behaves identically to one launched directly from the shell. |
| 4 | **Exit-code propagation** — Return the raw error from `cmd.Run()` directly. The shell can inspect the `*exec.ExitError` exit code. |
| 5 | **404 error messages** — Mirror `export.go`: `errors.Is(err, ErrNotFound)` on `ListSecrets` → `"project not found"`; on `GetSecret` → `"project or key not found"`. |
| 6 | **No new dependencies** — Implementation uses only the Go standard library (`os`, `os/exec`) plus packages already present in `internal/cli`. |
| 7 | **Minimum args** — `cobra.MinimumNArgs(2)` is retained: at least a project name and a program name must be provided. |

