# Language
All three components of the project (server, TUI and CLI) will be in GOLang.

## Server
The server will use the standard library to expose REST endpoints.
Encrypted secrets will be stored in a simple file-based storage, with one file per project. Each file will contain the key (in clear text) and the encrypted value. The secrets will be provided already encrypted by the clients.

## TUI Client
The TUI client will be built using the `bubbletea` library (v2)

## CLI Client
The CLI client will be built using the `cobra` library. No additional dependencies are expected for the CLI client.

# Authentication
Authentication will be done using ssh key pairs. The server will store clients public keys in a standard `authorized_keys` file, and the clients will use their private keys to authenticate.
All the common types of ssh keys will be supported (RSA, ECDSA, Ed25519, etc...). The server will use the `golang.org/x/crypto/ssh` package to handle ssh authentication.

# Encryption
All encryption will be done client-side using a key derived from the user's ssh key pair. The server will never have access to the unencrypted data, and all data stored on the server will be encrypted.
All the common types of ssh keys will be supported for encryption (RSA, ECDSA, Ed25519, etc...). The clients will use the `golang.org/x/crypto/ssh` package to handle ssh key parsing and encryption.

# Testing

## Unit Tests
Each package must be covered by unit tests written using the Go standard `testing` package. Table-driven tests are preferred. Mocks and fakes should be kept in-package where possible.

## Integration Tests
Integration tests verify the communication between components (server ↔ CLI, server ↔ TUI) and are orchestrated via Docker containers. A `docker-compose.yml` file under `tests/integration/` will spin up a real server instance plus any auxiliary containers needed. These tests are **semi-manual**: they are run explicitly by the developer (not in CI by default) using a dedicated `make integration-test` target or equivalent shell script. Each integration test scenario is documented as a step-by-step runbook in `tests/integration/README.md`.
