# EKVS Roadmap

> **Testing policy**
> - **Unit tests**: every milestone must ship with `*_test.go` files covering the new packages, written with the Go standard `testing` package (table-driven style).
> - **Integration tests**: semi-manual, Docker-based. Run explicitly via `make integration-test`. Scenarios are documented in `tests/integration/README.md`.

## Milestones

### Phase 0 — Project Foundation

1. **project_setup**
   Set up Go module, directory structure, shared packages (errors, config, logging), and CI skeleton. Include `Makefile` with `test` (unit) and `integration-test` targets.

2. **ssh_auth_primitives**
   Implement shared SSH key parsing, signature generation/verification, and key fingerprinting using `golang.org/x/crypto/ssh`. This package is a dependency of both server and clients.

3. **encryption_primitives**
   Implement client-side encryption/decryption helpers: key derivation from SSH key pair, symmetric encryption (AES-GCM), encode/decode of ciphertext blobs. All common SSH key types must be supported.

---

### Phase 1 — Server

4. **server_storage**
   Implement file-based storage engine: one file per project, key stored in clear text, value stored as encrypted blob (already encrypted by client). CRUD operations for projects and key-value pairs.

5. **server_auth**
   Implement SSH public key authentication for the REST API: `authorized_keys` management (register/list/revoke), per-request signature verification middleware.

6. **server_projects_api**
   REST endpoints for project management: create, list, delete projects per user.

7. **server_secrets_api**
   REST endpoints for secret management within a project: set, get, list, delete key-value pairs.

---

### Phase 2 — CLI Client

8. **cli_setup**
   Scaffold CLI with `cobra`: root command, global flags (server URL, identity file), config file loading.

9. **cli_auth**
   Commands to register a public key with the server and sign API requests using the user's private SSH key.

10. **cli_encryption**
    Integrate encryption primitives into the CLI: encrypt values before sending, decrypt values after receiving.

11. **cli_projects**
    Commands: `project create`, `project list`, `project delete`.

12. **cli_secrets**
    Commands: `secret set`, `secret get`, `secret list`, `secret delete`.

---

### Phase 3 — TUI Client

13. **tui_setup**
    Scaffold TUI with `bubbletea` v2: application entry point, navigation model, theme/styles.

14. **tui_auth**
    TUI flows for server registration and SSH key selection; sign requests transparently during the session.

15. **tui_encryption**
    Integrate encryption primitives into the TUI session context.

16. **tui_projects**
    Project list screen: browse, create, and delete projects interactively.

17. **tui_secrets**
    Secret management screen: list keys, view/copy decrypted values, add/edit/delete secrets.

18. **tui_ux_polish**
    Loading indicators, error modals, keyboard shortcut help, and overall UX refinement.

---

### Phase 4 — Integration Testing

19. **integration_test_setup**
    Create `tests/integration/` directory with `docker-compose.yml` (server container + client containers), `Makefile` target `integration-test`, and `README.md` runbook skeleton.

20. **integration_test_server_cli**
    Docker-based semi-manual integration scenarios covering server ↔ CLI communication: key registration, project CRUD, secret set/get/list/delete, encryption round-trip verification.

21. **integration_test_server_tui**
    Docker-based semi-manual integration scenarios covering server ↔ TUI communication: same flows as CLI scenarios but driven through the TUI interface.

