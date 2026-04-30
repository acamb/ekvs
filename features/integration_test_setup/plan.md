# Plan: integration_test_setup

Ordered list of tasks to implement the integration test environment.

---

## Tasks

### 1. Create multi-stage `Dockerfile`

File: `tests/integration/Dockerfile`

Stages:

| Stage | Base | Purpose |
|-------|------|---------|
| `keygen-nopass` | `alpine` | `ssh-keygen -t ed25519 -N ""` → `/keys/` |
| `keygen-passphrase` | `alpine` | `ssh-keygen -t ed25519 -N "changeme"` → `/keys/` |
| `server-build` | `golang:1.24-alpine` | `go build -o /server ./cmd/server` |
| `cli-build` | `golang:1.24-alpine` | `go build -o /cli ./cmd/cli` |
| `tui-build` | `golang:1.24-alpine` | `go build -o /tui ./cmd/tui` |
| `server-nopass` | `alpine` | binary + `ekvs.yaml` + public key from `keygen-nopass` |
| `server-passphrase` | `alpine` | binary + `ekvs.yaml` + public key from `keygen-passphrase` |
| `cli-nopass` | `alpine` | binary + private key from `keygen-nopass` |
| `cli-passphrase` | `alpine` | binary + private key from `keygen-passphrase` |
| `tui-nopass` | `alpine` | binary + private key from `keygen-nopass` + `ekvs-tui.yaml` |
| `tui-passphrase` | `alpine` | binary + private key from `keygen-passphrase` + `ekvs-tui.yaml` |

Server `ekvs.yaml` (same for both):
```yaml
server_addr:  "0.0.0.0:8080"
storage_path: "/data"
keys_dir:     "/data/.keys"
log_level:    "debug"
```

TUI `ekvs-tui.yaml` (same for both, path differs per image):
```yaml
profiles:
  - name:          "integration"
    server_url:    "http://server:8080"
    identity_file: "/root/.ssh/id_ed25519"
    theme:         "adaptive"
```

Public key copied as `/data/.keys/testuser.pub` (mode 0644).
Private key copied as `/root/.ssh/id_ed25519` (mode 0600).

---

### 2. Create `docker-compose.nopass.yml`

File: `tests/integration/docker-compose.nopass.yml`

Three services — all built from `tests/integration/Dockerfile` via `context: ../..`:

- `server` — `target: server-nopass`, command: `/server`, port `8080:8080`.
- `cli` — `target: cli-nopass`, `depends_on: [server]`, `command: sleep infinity`.
- `tui` — `target: tui-nopass`, `depends_on: [server]`, `command: sleep infinity`.

---

### 3. Create `docker-compose.passphrase.yml`

File: `tests/integration/docker-compose.passphrase.yml`

Same structure as above, with `target: server-passphrase`, `target: cli-passphrase`, `target: tui-passphrase`.

---

### 4. Update `Makefile`

Replace the existing `integration-test` target:

```makefile
## integration-test: start nopass scenario (server + cli + tui)
integration-test:
	cd tests/integration && docker compose -f docker-compose.nopass.yml up --build -d

## integration-test-passphrase: start passphrase scenario (server + cli + tui)
integration-test-passphrase:
	cd tests/integration && docker compose -f docker-compose.passphrase.yml up --build -d

## integration-test-down: stop all running integration containers
integration-test-down:
	cd tests/integration && docker compose -f docker-compose.nopass.yml down 2>/dev/null; docker compose -f docker-compose.passphrase.yml down 2>/dev/null; true
```

---

### 5. Create `tests/integration/README.md`

Sections:
- **Prerequisites**: Docker + Docker Compose v2, `make`.
- **Directory layout**: what each file is for.
- **Key strategy**: ed25519 key pair baked in at build time; public key pre-registered in server; private key in cli/tui containers at `/root/.ssh/id_ed25519`; keys never touch the repo working tree. Passphrase for the passphrase scenario: `changeme`.
- **Running a scenario**: `make integration-test` (nopass) or `make integration-test-passphrase`.
- **Attaching to a container**:
  ```bash
  docker exec -it integration-tui-1 /tui
  docker exec -it integration-cli-1 /cli --help
  ```
- **Stopping**: `make integration-test-down`.
