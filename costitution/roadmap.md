# EKVS Roadmap

> **Testing policy**
> - **Unit tests**: every milestone must ship with `*_test.go` files covering the new packages, written with the Go standard `testing` package (table-driven style).
> - **Integration tests**: semi-manual, Docker-based. Run explicitly via `make integration-test`. Scenarios are documented in `tests/integration/README.md`.

## Milestones

### Phase 0 — Project Foundation

1. **project_setup**
   Set up Go module, directory structure, shared packages (errors, config, logging). Include `Makefile` with `test` (unit) and `integration-test` targets.

2. **ssh_auth_primitives**
   Implement shared SSH key parsing, signature generation/verification, and key fingerprinting using `golang.org/x/crypto/ssh`. This package is a dependency of both server and clients.

3. **encryption_primitives**
   Implement client-side encryption/decryption helpers: key derivation from SSH key pair, symmetric encryption (AES-GCM), encode/decode of ciphertext blobs. All common SSH key types must be supported.

---

### Phase 1 — Server

4. **server_storage**
   Implement file-based storage engine: one file per project, key stored in clear text, value stored as encrypted blob (already encrypted by client). CRUD operations for projects and key-value pairs.

5. **server_auth**
   Implement SSH public key authentication for the REST API. Public keys are managed by the system administrator by placing `.pub` files (OpenSSH authorized_keys format, free-form filename) in `{storage_root}/.keys/`; there is no HTTP API for key registration or revocation. Per-request signature verification middleware reads `X-Timestamp`, `X-Fingerprint` and `X-Signature` headers, verifies the signature against the stored key, and enforces a **30-second replay-protection window** (consistent with `ssh_auth_primitives.CheckTimestamp`).
   > **Contract with `server_storage`**: the `userID` passed to any `Store` method must be the canonical SSH fingerprint string produced by `golang.org/x/crypto/ssh.FingerprintSHA256`, e.g. `SHA256:<base64>`. The store sanitises this value for filesystem use but does **not** guarantee isolation if two distinct fingerprints produce the same sanitised path. `server_auth` is responsible for ensuring every authenticated request carries the correct canonical fingerprint.

6. **server_projects_api**
   REST endpoints for project management: create, list, delete projects per user.

7. **server_config**
   Load server configuration from a YAML file (default: `ekvs.yaml`) whose path can be overridden via a `--config` CLI flag. Environment variables continue to work as overrides (env > file > defaults). On startup the server automatically creates any missing directories (`StoragePath`, `KeysDir`). The `internal/config` package is updated to support YAML loading; `cmd/server/main.go` gains flag parsing.

8. **server_secrets_api**
   REST endpoints for secret management within a project: set, get, list, delete key-value pairs.

9. **code cleanup**
   Refactor and clean up server codebase: remove any temporary scaffolding, ensure consistent error handling and logging, resolve warnings.

---

### Phase 2 — TUI Client

10. **tui_setup**
    Scaffold TUI with `bubbletea` v2: application entry point, navigation model, theme/styles.

11. **tui_auth**
    TUI flows for SSH key selection; sign requests transparently during the session.

12. **tui_projects**
    Project list screen: browse, create, and delete projects interactively.

13. **tui_encryption**
    Integrate encryption primitives into the TUI session context.

14. **tui_secrets**
    Secret management screen: list keys, view/copy decrypted values, add/edit/delete secrets.

15. **tui_profiles**
    Profile management screen: create a new connection profile (server URL, SSH identity file, theme), edit the fields of the currently active profile, delete an existing profile, and **switch to a different profile**. Changes are persisted to the YAML configuration file. Deleting the active profile redirects the user to the profile selection screen (or the first-run wizard if no profiles remain). **Switching to a different profile clears all session state (loaded private key, passphrase, fingerprint) before loading the new profile**, then re-triggers the auth flow the next time an authenticated operation is requested.

16. **tui_ux_polish**
    Loading indicators, error modals, keyboard shortcut help, and overall UX refinement.

17. **tui_e2e_tests**
    End-to-end tests for TUI screens using [`teatest`](https://github.com/charmbracelet/x/tree/main/exp/teatest) (the official Bubble Tea v2 test harness). Each TUI screen (`projects`, `secrets`, `profiles`, `auth`, `wizard`, `mainModel`) gets a `*_e2e_test.go` file that drives the model via simulated key presses and asserts on the rendered output. Covers happy paths and error paths (e.g. spinner appears on load, modal appears on error, footer hints match the current mode). The `teatest` package is added to `go.mod` as a test-only dependency.

---

### Phase 3 — CLI Client

18. **cli_setup**
    Scaffold CLI with `cobra`: root command, global flags (server URL, identity file), config file loading.

19. **cli_auth**
    Commands to sign API requests using the user's private SSH key.

20. **cli_encryption**
    Integrate encryption primitives into the CLI: encrypt values before sending, decrypt values after receiving.

21. **cli_projects**
    Commands: `project create`, `project list`, `project delete`.

22. **cli_secrets**
    Commands: `secret set`, `secret get`, `secret list`, `secret delete`.

---

### Phase 4 — Integration Testing

23. **integration_test_setup**
    Create `tests/integration/` directory with `docker-compose.yml` (server container + client containers), `Makefile` target `integration-test`, and `README.md` runbook skeleton.

24. **integration_test_server_cli**
    Docker-based semi-manual integration scenarios covering server ↔ CLI communication: key registration, project CRUD, secret set/get/list/delete, encryption round-trip verification.

25. **integration_test_server_tui**
    Docker-based semi-manual integration scenarios covering server ↔ TUI communication: same flows as CLI scenarios but driven through the TUI interface.

---

### Phase 5 — CI Pipeline

26. **ci_pipeline**
    GitHub Actions workflow that runs on every push/PR to `main`: checkout, setup-go, `make test`, `make lint`. Integration tests are explicitly excluded from CI and remain manual only.

---

### Phase 6 — SSH Agent Support

27. **ssh_agent_support**
    Add opt-in support for signing via the SSH agent (`SSH_AUTH_SOCK`). When the agent is available and the configured `identity_file` public key is present in the agent, both TUI and CLI clients will delegate signing to the agent instead of loading the private key from disk, so no passphrase prompt is needed. Falls back to the current file-based flow if the agent is unavailable or does not hold the required key. The `internal/ssh` package is extended with an `AgentSigner` helper; TUI and CLI auth flows are updated to probe the agent first.

