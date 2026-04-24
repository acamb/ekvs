# plan.md — project_setup

## Ordered Task List

1. **Init Go module**
   Confirm/update `go.mod` with the canonical module path (e.g. `github.com/<org>/ekvs`) and minimum Go version. Run `go mod tidy`.

2. **Create directory layout**
   Scaffold the full directory tree as defined in `requirements.md`. Create empty `.gitkeep` files where needed so the structure is committed.

3. **Add `.gitignore` entries**
   Add Go-standard ignores (`/bin/`, `*.test`, `coverage.out`) plus project-specific ones (`/dist/`, integration test artefacts).

4. **Scaffold `internal/errors` package**
   Create `internal/errors/errors.go` with sentinel error types and a custom `AppError` type. Add `internal/errors/errors_test.go` with table-driven tests.

5. **Scaffold `internal/config` package**
   Create `internal/config/config.go` exposing a `Config` struct and a `Load() (*Config, error)` function (reads from env/file). Add `internal/config/config_test.go`.

6. **Scaffold `internal/logging` package**
   Create `internal/logging/logging.go` exposing a `Logger` interface and a `New(level string) Logger` constructor backed by `log/slog`. Add `internal/logging/logging_test.go`.

7. **Write `Makefile`**
   Add targets: `build`, `test`, `integration-test`, `lint`, `clean`. `test` runs `go test ./...`; `integration-test` delegates to `tests/integration/` (stub for now).

8. **Create integration test skeleton**
   Add `tests/integration/README.md` (runbook placeholder) and `tests/integration/docker-compose.yml` (stub with a commented-out server service). Add the `integration-test` Makefile target pointing to a stub script.

9. **Verify and tidy**
   Run `go mod tidy`, `make test`, `make lint` locally; confirm all pass with the scaffolded (empty) packages.


