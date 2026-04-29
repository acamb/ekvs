# cli_setup — Plan

1. **Add cobra dependency**
   Verify `cobra` is present in `go.mod`; if not, run `go get github.com/spf13/cobra` and update `go.sum`.

2. **Create `internal/cli/root.go`**
   Define the `rootCmd` cobra command. Declare `--server` and `--identity` persistent flags. On `PersistentPreRunE` resolve each flag from env var if not set via flag; return an error if still empty.

3. **Create `internal/cli/export.go`**
   Register `export` subcommand under `rootCmd`. Accept positional args `<projectName> [keyName]`. Print "not yet implemented" and return `ErrNotImplemented` (exit 1).

4. **Create `internal/cli/exec.go`**
   Register `exec` subcommand under `rootCmd`. Accept positional args `<projectName> [keyName] -- <program> [args]`. Print "not yet implemented" and return `ErrNotImplemented` (exit 1).

5. **Wire `cmd/cli/main.go`**
   Call `cli.Execute()` (exported wrapper around `rootCmd.Execute()`). Propagate non-nil errors to `os.Exit(1)`.

6. **Unit tests — `internal/cli/cli_test.go`**
   Table-driven tests using `cobra`'s `SetArgs` + output capture:
   - Flag precedence: flag beats env var.
   - Missing `--server` → error.
   - Missing `--identity` → error.
   - `export` with no args → usage error (cobra enforces `MinimumNArgs(1)`).
   - `exec` with no args → usage error.

