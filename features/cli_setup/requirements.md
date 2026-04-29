# cli_setup — Requirements

## Scope
Scaffold the CLI client using the `cobra` library. This milestone produces a working binary skeleton with global flags, environment variable support, and a stub subcommand tree. No real API calls are made yet; actual auth, encryption, and commands are added in subsequent milestones.

## Global flags
Both flags must also be readable from environment variables (flag takes precedence over env var):

| Flag | Env var | Description |
|------|---------|-------------|
| `--server` | `EKVS_SERVER` | Server address in `host:port` form. Required for any subcommand. |
| `--identity` | `EKVS_IDENTITY` | Path to the OpenSSH private key file. Required for any subcommand. |

Missing required flags produce a clear error message and a non-zero exit code before any network call is attempted.

## Command structure (flat)
```
ekvs [global flags] <command> [args]
```

Commands planned across all CLI milestones:
- `export <projectName> [keyName]` — decrypt and print secrets
- `exec <projectName> [keyName] -- <program> [args]` — inject secrets as env vars

This milestone only scaffolds the root command and registers stub subcommands (`export`, `exec`) that print "not yet implemented" and exit 1.

## Configuration
No config file. All configuration is via flags and env vars only.

## Binary entry point
`cmd/cli/main.go` — already present in the repo; it must be wired to the cobra root command.

## Package layout
```
internal/cli/
    root.go       — root cobra command, global flags, env var resolution
    export.go     — export subcommand stub
    exec.go       — exec subcommand stub
```

## Decisions
- No config file (flags + env vars only).
- No SSH agent support in this phase.
- Cobra v1 (already a transitive dep; confirm in go.mod, add explicitly if absent).
- Output always goes to `os.Stdout`; errors/logs to `os.Stderr`.

