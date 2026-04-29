# cli_setup — Validation

## Build check
```bash
go build ./cmd/cli/...
```
Binary must compile without errors.

## Unit tests
```bash
go test ./internal/cli/...
```
All table-driven tests must pass.

## Manual smoke tests

### Missing flags
```bash
ekvs export myproject
# expected: error about missing --server and/or --identity, exit 1
```

### Flag takes precedence over env var
```bash
export EKVS_SERVER=env-host:8080
ekvs --server flag-host:9090 --identity ~/.ssh/id_ed25519 export myproject
# expected: "not yet implemented", exit 1 (no error about missing flags)
```

### Env var fallback
```bash
export EKVS_SERVER=localhost:8080
export EKVS_IDENTITY=~/.ssh/id_ed25519
ekvs export myproject
# expected: "not yet implemented", exit 1
```

### Stub subcommands
```bash
ekvs --server localhost:8080 --identity ~/.ssh/id_ed25519 export myproject
# expected: "not yet implemented", exit 1

ekvs --server localhost:8080 --identity ~/.ssh/id_ed25519 exec myproject -- env
# expected: "not yet implemented", exit 1
```

### Help output
```bash
ekvs --help
ekvs export --help
ekvs exec --help
```
All must print usage without errors.

