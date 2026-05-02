# EKVS - Easy Key Value Store

This project aims to provide a simple way to store and inject secrets into applications and containers, the principal use case
is injecting secrets into docker compose or starting applications with an environment populated with secrets fetched from EKVS.

## Overview
EKVS is a simple, secure, self-hosted key-value store designed for managing secrets.
It consists of 3 main components:
 * server, a REST API server that handles authentication, project and secret storage.
 * tui, a terminal-based interactive client built with Bubble Tea to manage secrets.
 * cli, a read-only command-line client to fetch, decrypt and print/export secrets to an environment.

## Features

 * Secrets are stored encrypted and can be decrypted only by the clients.
 * Encryption key is derived from clients private keys and never leaves the client side.
 * Authentication is done via SSH keys using a traditional authorized_keys style approach. Each key is a user.
 * Projects are namespaces for secrets, users can have multiple projects and secrets.

## Getting Started
The server is distributed as a single binary or a Docker image.
Mount a local directory as `/data` to persist projects, secrets and keys.
Put the public key of your SSH key pair in `/data/.keys/` to allow clients to authenticate.
### Running the Server with Docker
Using docker run:
```bash
docker run -d \
    -p 8080:8080 \
    -v /path/to/storage:/data \
    --name ekvs-server \
    acamb23/ekvs:latest
```
Using docker compose:
```yaml
services:
  ekvs-server:
    image: acamb23/ekvs:latest
    ports:
      - "8080:8080"
    volumes:
      - /path/to/storage:/data
```

### Populate the store with the TUI Client
The TUI client can be used to create projects and secrets in an interactive way.
"Projects" are namespaces for secrets, and "Secrets" are key-value pairs stored encrypted in the server.
A SSH key pair is a user, so you need to use the same key on the TUI and CLI to access the same projects and secrets.

### Retrieving Secrets with the CLI and launching Applications
The CLI client can be used to fetch secrets and launch applications with those secrets in the environment.
```bash
# Fetch secrets for a project and print them in KEY=VALUE format
cli --server http://localhost:8080 --identity /path/to/private/key print project_name
# Fetch a single secret value
cli --server http://localhost:8080 --identity /path/to/private/key print project_name secret_name
# Save a secret value to a file 
cli --server http://localhost:8080 --identity /path/to/private/key print project_name secret_name --output /path/to/file
# Launch a commandpopulating the environment with secrets
cli --server http://localhost:8080 --identity /path/to/private/key exec project_name -- /bin/sh -c "export"
# Launch docker compose with secrets for a project in the environment
cli --server http://localhost:8080 --identity /path/to/private/key exec project_name -- docker compose up
```