# Requirements: integration_test_setup

## Goal

Provide a fully self-contained Docker-based environment for **manual** integration testing of the EKVS server, CLI and TUI clients. The environment must be startable with a single command per scenario and require no manual configuration steps.

## Scope

- Docker images for `server`, `cli`, and `tui` binaries.
- SSH keys generated at **build time** (baked into images); no runtime key generation.
- **Empty** storage: no pre-loaded projects or secrets.
- Two independent scenarios, each with its own `docker-compose` file, differentiated by SSH key type:
  - **nopass scenario** (`docker-compose.nopass.yml`): ed25519 key with **no passphrase**.
  - **passphrase scenario** (`docker-compose.passphrase.yml`): ed25519 key **protected by a passphrase**.
- Each scenario spins up **three containers**: `server`, `cli`, `tui`.
- `cli` and `tui` containers stay running idle (`sleep infinity`) so the developer can attach on demand.
- TUI and CLI are launched manually by the developer via `docker exec -it <container> /tui` / `docker exec -it <container> /cli`.
- `Makefile` updated with per-scenario targets.
- `tests/integration/README.md` documenting the environment structure.

## Out of Scope

- Automated test assertions / scripts (scenarios are documented runbooks, not automated checks).
- CI execution (integration tests remain manual-only).
- Pre-seeded data.

## Key Decisions

| # | Decision |
|---|----------|
| 1 | Two `docker-compose` files, one per SSH key scenario (`nopass`, `passphrase`). |
| 2 | Single multi-stage `Dockerfile` with named stages; each scenario uses dedicated build `target:`s that bake in the appropriate key variant. |
| 3 | SSH key pair (ed25519) generated in a `keygen` build stage; public key placed in `/data/.keys/` in the server image; private key placed at `/root/.ssh/id_ed25519` in cli and tui images. |
| 4 | Passphrase scenario: same structure, key generated with a fixed known passphrase (`changeme`) documented in README. |
| 5 | TUI image includes a baked-in `ekvs-tui.yaml` pointing to `http://server:8080` using `/root/.ssh/id_ed25519`. |
| 6 | `cli` and `tui` containers use `command: sleep infinity` so they stay up without running the application. |
| 7 | `make integration-test` starts the nopass scenario; `make integration-test-passphrase` starts the passphrase scenario. |
| 8 | Generated keys are never written to the repo working tree; they exist only inside Docker build layers. |

## Constraints

- Images must build from the repo root context (all `go build` calls reference `./cmd/…`).
- Server listens on `0.0.0.0:8080` inside the container; docker-compose service name `server` is used as hostname by clients.
- Base images: `golang:1.24-alpine` (builder), `alpine:latest` (runtime).
- Fixed passphrase for the passphrase scenario: `changeme` (for testing only).
