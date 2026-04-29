# requirements.md — cli_encryption

## Goal

Wire `internal/encryption` into the CLI layer by introducing a `Session` struct
in `internal/cli/session.go`. The session exposes a `Decrypt` method that
`cli_export` and `cli_exec` will consume. No command logic is implemented here.

---

## Scope

### In scope
- Create `internal/cli/session.go`:
  - `Session` struct with unexported `encKey []byte` field.
  - `NewSession(signer crypto.Signer) (*Session, error)` — calls `encryption.DeriveKey` and caches the key.
  - `Decrypt(encoded string) (string, error)` method — delegates to `encryption.Decrypt`; wraps errors in `ErrDecrypt`.
  - Package-level error sentinels: `ErrDecrypt`, `ErrNotAuthenticated`.
- Unit tests in `internal/cli/session_test.go` (table-driven).
- Documentation comment in `root.go` `PersistentPreRunE` explaining that session creation is per-command (lazy), not global.

### Out of scope
- `cli_export` and `cli_exec` command implementations.
- `Encrypt` method (CLI is read-only with respect to secret values).
- Key rotation or re-encryption.
- SSH agent support.

---

## Decisions

| # | Decision |
|---|----------|
| 1 | `Session` lives in `internal/cli/session.go` — same package as the cobra commands, avoiding an extra sub-package. |
| 2 | Key derivation is **eager**: it happens in `NewSession`, not lazily on first `Decrypt` call. |
| 3 | `Decrypt` wraps raw `internal/encryption` errors in `ErrDecrypt` so commands do not need to import `internal/encryption`. |
| 4 | Session creation is **per-command** (lazy): `root.go` `PersistentPreRunE` does NOT call `NewSession`; each command that needs decryption calls `NewSession` itself after loading the identity. |
| 5 | `encKey` is unexported; consumers interact only through `Decrypt`. |
| 6 | CLI does not expose an `Encrypt` method because the CLI never writes secret values (write path is TUI only). |

