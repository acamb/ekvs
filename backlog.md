# Backlog

Items not scheduled in the current roadmap. They will be prioritised and inserted in the roadmap in a future planning session.

---

## Security & Authentication

### `passphrase_protected_keys`
Support SSH private keys protected by a passphrase in both CLI and TUI clients.
The client should prompt the user for the passphrase at key-load time and never store it in memory longer than necessary.

### `ssh_agent_support`
Allow CLI and TUI clients to delegate signing (and optionally key-derivation for encryption) to a running SSH agent via the `SSH_AUTH_SOCK` socket, using `golang.org/x/crypto/ssh/agent`. This removes the need for the client to ever touch the raw private key file.

