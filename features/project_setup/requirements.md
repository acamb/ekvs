# requirements.md — project_setup

## Scope

### In scope
- Go module initialisation and canonical module path.
- Full directory tree scaffolding (all top-level and internal packages).
- Three shared packages: `internal/errors`, `internal/config`, `internal/logging` — types and interfaces only, no business logic.
- `Makefile` with `build`, `test`, `integration-test`, `lint`, `clean` targets.
- `.gitignore` additions.
- Integration test directory skeleton (`docker-compose.yml` stub + `README.md` placeholder).

### Out of scope
- Any SSH, encryption, storage, or API logic.
- Full implementation of config file parsing beyond a basic `Load()` stub.
- Real Docker containers or runnable integration scenarios.
- CLI or TUI scaffolding.
- GitHub Actions CI pipeline (covered in the dedicated `ci_pipeline` milestone).

---

## Directory Structure

```
ekvs/
├── cmd/
│   ├── server/
│   │   └── main.go          # stub: package main, func main() {}
│   ├── tui/
│   │   └── main.go          # stub
│   └── cli/
│       └── main.go          # stub
├── internal/
│   ├── errors/
│   │   ├── errors.go
│   │   └── errors_test.go
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   └── logging/
│       ├── logging.go
│       └── logging_test.go
├── tests/
│   └── integration/
│       ├── docker-compose.yml
│       └── README.md
├── .gitignore
├── go.mod
├── go.sum
└── Makefile
```

---

## Shared Packages

### `internal/errors`
- `type AppError struct` — wraps an underlying error with a human-readable `Code string` and `Message string`.
- `func (e *AppError) Error() string`
- `func (e *AppError) Unwrap() error`
- Sentinel values: `ErrNotFound`, `ErrUnauthorized`, `ErrInternal`.

### `internal/config`
- `type Config struct` — fields: `ServerAddr string`, `StoragePath string`, `LogLevel string`.
- `func Load() (*Config, error)` — reads from environment variables; returns defaults if not set.

### `internal/logging`
- `type Logger interface` — methods: `Info(msg string, args ...any)`, `Error(msg string, args ...any)`, `Debug(msg string, args ...any)`.
- `func New(level string) Logger` — returns a concrete logger backed by `log/slog`.

---

## Makefile Targets

| Target             | Command                                      | Notes                              |
|--------------------|----------------------------------------------|------------------------------------|
| `build`            | `go build ./cmd/...`                         | Compiles all three binaries        |
| `test`             | `go test ./... -race -count=1`               | Runs all unit tests                |
| `integration-test` | `cd tests/integration && docker compose up`  | Semi-manual, not run in CI         |
| `lint`             | `go vet ./...`                               | Static analysis                    |
| `clean`            | `rm -rf bin/ coverage.out`                   | Removes build artefacts            |

---

## Open Decisions

None. All choices are consistent with the tech stack and roadmap defined in `costitution/`.




