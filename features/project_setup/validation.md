# validation.md — project_setup

## Manual Checklist

- [ ] `go.mod` contains the correct module path and Go version ≥ 1.22.
- [ ] `go.sum` exists and is not empty after `go mod tidy`.
- [ ] All directories listed in the tree in `requirements.md` exist and are committed.
- [ ] Each `cmd/*/main.go` compiles as a valid `package main` with an empty `main()`.
- [ ] `internal/errors`, `internal/config`, `internal/logging` each have a `.go` source file and a `_test.go` file.
- [ ] `.gitignore` includes at minimum: `bin/`, `*.test`, `coverage.out`, `dist/`.
- [ ] `Makefile` exists at repo root with all five targets defined.
- [ ] `tests/integration/docker-compose.yml` and `tests/integration/README.md` both exist.

---

## Running Unit Tests

```zsh
make test
# equivalent to: go test ./... -race -count=1
```

**"Passing" at this stage means:**
- Exit code `0`.
- All three `internal/*` packages are found and their minimal table-driven tests execute without failure.
- No data races detected (`-race`).
- Output shows:
  ```
  ok  ekvs/internal/errors
  ok  ekvs/internal/config
  ok  ekvs/internal/logging
  ```
- Stub `cmd/` packages produce no test output (no `_test.go` files) — this is acceptable.

---

## Verifying Makefile Targets

```zsh
# Unit tests
make test          # must exit 0

# Build all binaries
make build         # must exit 0

# Lint
make lint          # must exit 0 with no vet errors

# Clean
make clean         # must exit 0; bin/ and coverage.out removed afterwards
```zsh
# Integration test skeleton (stub only)
make integration-test   # must exit 0; prints an informational message or docker compose output
```



