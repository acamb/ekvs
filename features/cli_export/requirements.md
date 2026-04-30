# requirements.md — cli_export

## Goal

Implement `ekvs export <projectName> [keyName]`: fetch encrypted secret(s) from the
server, decrypt them client-side using the SSH-derived session key, and print the
result to stdout in `KEY=value` format (one line per secret).

---

## Scope

### In scope

- `internal/cli/client.go`:
  - `secretEntry` struct (`Key`, `Value string`).
  - `ErrNotFound` sentinel error returned on HTTP 404.
  - `Client` struct (`BaseURL string`, `HTTPClient *http.Client`) with `NewClient`.
  - `ListSecrets` — calls `GET /projects/{name}/secrets`.
  - `GetSecret` — calls `GET /projects/{name}/secrets/{key}`.
  - Every request is signed via `SignedHeaders`.
- `internal/cli/export.go` — replace stub; implement the two paths (single key, all keys).
- Unit tests in `internal/cli/export_test.go` (package `cli_test`).

### Out of scope

- `exec.go` implementation (separate feature `cli_exec`).
- HTTPS / TLS support (considered a future concern).
- Sorting of output beyond API-returned order.
- Key creation / update / deletion via the CLI.

---

## Decisions

| # | Decision |
|---|----------|
| 1 | `BaseURL` in `Client` is always `http://` + `flagServer` (`host:port`). No `--tls` flag for now. |
| 2 | `ErrNotFound` is returned as-is from `RunE`; cobra prints it and exits with code 1. |
| 3 | Output order is exactly the order returned by the server API (no extra sorting). |
| 4 | `HTTPClient` field defaults to `http.DefaultClient`; tests inject an `httptest` server URL instead. |
| 5 | `secretEntry` is defined in `client.go` (package `cli`), not re-exported. `export.go` uses it directly. |

