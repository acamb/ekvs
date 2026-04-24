# Integration Tests — Runbook

Integration tests are **semi-manual** and require Docker + Docker Compose.

## Prerequisites

- Docker Engine ≥ 24
- Docker Compose v2 (`docker compose` subcommand)
- A valid SSH key pair for test users

## Running

```zsh
make integration-test
```

This command starts the services defined in `docker-compose.yml` and streams
their logs. Press `Ctrl-C` to stop.

## Scenarios

> Scenarios will be added as the server and client milestones are completed.

| # | Scenario | Components | Status |
|---|----------|------------|--------|
| — | *(none yet)* | — | — |

