# plan.md — cli_export_file

Ordered task list. Each task must pass `make test` before the next begins.

---

## Task 1 — Register `--output` flag in `internal/cli/export.go`

**File**: `internal/cli/export.go`

- Declare a package-level `flagOutput string` variable.
- In the `init()` function (alongside the other persistent flags), call:
  `exportCmd.Flags().StringVar(&flagOutput, "output", "", "write decrypted value to file (single key only)")`
- Flag is local (`Flags()`, not `PersistentFlags()`).

---

## Task 2 — Add guard validation in `RunE`

**File**: `internal/cli/export.go`

At the top of `RunE`, before `LoadIdentity`, add:

```go
if flagOutput != "" && len(args) != 2 {
    return fmt.Errorf("--output requires a keyName argument")
}
```

This fires before any identity load or network call.

---

## Task 3 — Extend the single-key branch to write to file

**File**: `internal/cli/export.go`

Inside the `len(args) == 2` branch, after `session.Decrypt` succeeds, branch on `flagOutput`:

- **`flagOutput == ""`**: keep existing `fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", ...)`.
- **`flagOutput != ""`**: call `os.WriteFile(flagOutput, []byte(plaintext), 0600)`; on error
  return `fmt.Errorf("write output file: %w", err)`.

The all-keys branch (`len(args) == 1`) is not modified.

---

## Task 4 — Add unit tests in `internal/cli/export_test.go`

**File**: `internal/cli/export_test.go`

Add a `runExportToFile` helper analogous to `runExport` that appends `--output <path>` to the
args slice passed to `ExecuteWithArgs`.

Implement the tests listed in `validation.md § Required tests`.

---

## Task 5 — Final check

- `go test -count=1 ./internal/cli/...` passes with no failures.
- `go build ./...` succeeds.
- Manual smoke-tests per `validation.md §§ 3–5`.

