# Integration Test Environment

Manual integration test environment for EKVS. Spins up three containers (`server`, `cli`, `tui`) pre-configured to communicate with each other using a baked-in SSH key pair.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) with Compose v2 (`docker compose`)
- `make`

## Directory Layout

```
tests/integration/
├── Dockerfile                    # Multi-stage build: keygen → server/cli/tui per scenario
├── docker-compose.nopass.yml     # Scenario: SSH key without passphrase
├── docker-compose.passphrase.yml # Scenario: SSH key protected by passphrase
└── README.md                     # This file
```

## SSH Key Strategy

An ed25519 key pair is generated **at image build time** inside the `keygen-*` Docker build stages via `ssh-keygen`. The keys are never written to the repository working tree — they exist only inside Docker build layers.

| Artifact | Location in container | Description |
|---|---|---|
| Public key | `/data/.keys/testuser.pub` (server) | Pre-registered; server accepts requests signed with the matching private key |
| Private key | `/root/.ssh/id_ed25519` (cli, tui) | Used to sign API requests and derive the encryption key |

The `cli` and `tui` containers also have the environment variables `EKVS_SERVER` and `EKVS_IDENTITY` pre-set, so no flags are needed when invoking the binaries manually.

## Scenarios

### Nopass — SSH key without passphrase

```bash
make integration-test
```

### Passphrase — SSH key protected by passphrase

```bash
make integration-test-passphrase
```

> **Passphrase**: `changeme` (for testing only)

Both commands start all three containers in the background (`-d`). The `cli` and `tui` containers remain idle (`sleep infinity`) until you attach to them manually.

## Attaching to Containers

Wait for the containers to be healthy, then exec into whichever you need:

```bash
# Launch the TUI interactively
docker exec -it integration-tui-1 /tui

# Open a shell in the CLI container and run commands
docker exec -it integration-cli-1 sh

# Example CLI commands (inside the cli container shell)
/cli projects list
/cli projects create myproject
/cli secrets set myproject MY_KEY "hello world"
/cli export myproject
/cli exec myproject -- env | grep MY_KEY
```

> Container names follow Docker Compose defaults: `integration-<service>-1`.
> If you used a different project name, run `docker compose ps` to find the actual names.

## Stopping

```bash
make integration-test-down
```

This stops and removes containers for both scenarios.

## Server Configuration

The server starts with:

```yaml
server_addr:  "0.0.0.0:8080"
storage_path: "/data"
keys_dir:     "/data/.keys"
log_level:    "debug"
```

Storage is **empty** on every fresh `up`. To persist data between restarts, add a named volume to the compose file.

## TUI Configuration

The TUI starts with a single pre-configured profile:

```yaml
profiles:
  - name:          "integration"
    server_url:    "http://server:8080"
    identity_file: "/root/.ssh/id_ed25519"
    theme:         "adaptive"
```
