# requirements.md — server_projects_api

## User Decisions

| Decisione | Scelta |
|-----------|--------|
| Router | `net/http` standard library + `http.ServeMux` (nessuna dipendenza esterna) |
| Versione Go | Go 1.25 (da `go.mod`); il routing con method nel pattern (`"METHOD /path/{name}"`) è disponibile nativamente da Go 1.22 |
| Base path | `/projects` — tutti gli endpoint sono sotto questo prefisso |
| Autenticazione | `auth.AuthMiddleware` avvolge il router; userID estratto da contesto via `auth.UserIDFromContext` |
| Formato risposta | JSON (`Content-Type: application/json`) per tutte le risposte, sia successo che errore |
| Corpo errori | `{"error": "<messaggio>"}` — coerente con `server_auth` |
| Nomi progetto | La validazione è **delegata a `internal/storage`** solo per `CreateProject`: `storage.ErrInvalidName` → `400 Bad Request`. Per `DeleteProject`, `storage` non valida il nome e un nome invalido restituisce semplicemente `404` (il file non può esistere). Il layer HTTP non replica la regex. |
| Conflitti | `CreateProject` su progetto già esistente → `409 Conflict` |
| Non trovato | `DeleteProject` su progetto inesistente → `404 Not Found` |
| Package path | `internal/server` |
| Configurazione | Tramite `internal/config`: `config.Load()` legge `EKVS_SERVER_ADDR`, `EKVS_STORAGE_PATH`, `EKVS_KEYS_DIR` (aggiunto in questa feature), `EKVS_LOG_LEVEL` |
| Logging | `internal/logging.Logger` iniettato in `ProjectsHandler`; errori interni (`500`) loggati con `.Error()`; richieste valide con `.Debug()` |
| Mounting | Il server HTTP principale (`cmd/server/main.go`) monta il router dei progetti e applica il middleware di auth |
| `GET /projects/{name}` | **Non implementato in questa feature** — sarà il punto di ingresso per `server_secrets_api` (milestone 7) |
| Test strategy | Storage reale su disco (`t.TempDir()`); mock solo se emergono casi non raggiungibili altrimenti |

---

## Scope

### In scope
- `ProjectsHandler(store *storage.Store, log logging.Logger) http.Handler` — restituisce un `http.Handler` che espone le rotte per la gestione dei progetti.
- Tre endpoint REST (vedi sezione API).
- Traduzione degli errori di `internal/storage` in codici HTTP appropriati.
- Unit tests (table-driven) con copertura ≥ 90% degli statement, usando storage reale su `t.TempDir()`.
- Aggiornamento di `cmd/server/main.go` per cablare config, storage, keystore, logger, middleware di auth e handler dei progetti.
- Aggiornamento di `internal/config` per aggiungere il campo `KeysDir` / variabile `EKVS_KEYS_DIR`.

### Out of scope
- `GET /projects/{name}` — live in `server_secrets_api`.
- Endpoint per la gestione dei segreti — live in `server_secrets_api`.
- Gestione delle chiavi SSH — live in `server_auth`.
- TLS / rate limiting.
- Paginazione dei risultati.

---

## Package Path

```
internal/server
```

Import path: `ekvs/internal/server`

---

## API

### `POST /projects/{name}`
Crea un nuovo progetto per l'utente autenticato.

**Request**: nessun corpo.

**Response successo**: `201 Created`
```json
{"name": "my-project"}
```

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Nome non valido (`storage.ErrInvalidName`) | `400 Bad Request` |
| Progetto già esistente (`storage.ErrProjectAlreadyExists`) | `409 Conflict` |
| Errore interno | `500 Internal Server Error` |

---

### `GET /projects`
Elenca tutti i progetti dell'utente autenticato.

**Request**: nessun corpo.

**Response successo**: `200 OK`
```json
{"projects": ["alpha", "beta", "gamma"]}
```
Lista ordinata alfabeticamente (delegata a `storage.ListProjects`). Se l'utente non ha progetti, ritorna lista vuota `[]`.

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Errore interno | `500 Internal Server Error` |

---

### `DELETE /projects/{name}`
Elimina un progetto dell'utente autenticato e tutti i segreti in esso contenuti.

**Request**: nessun corpo.

**Response successo**: `204 No Content` (nessun corpo).

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Progetto non trovato o nome invalido (`storage.ErrProjectNotFound`) | `404 Not Found` |
| Errore interno | `500 Internal Server Error` |

> **Decisione**: `storage.DeleteProject` non chiama `validateName`. Un nome
> sintatticamente invalido non può esistere su disco e produce `ErrProjectNotFound`.
> Il layer HTTP non pre-valida il nome per DELETE: nomi invalidi restituiscono `404`.

---

## Exported API

```go
package server

import (
    "net/http"

    "ekvs/internal/logging"
    "ekvs/internal/storage"
)

// ProjectsHandler returns an http.Handler that exposes project management
// endpoints. It must be wrapped by auth.AuthMiddleware before mounting.
// log is used to emit structured logs for internal errors and debug traces.
//
// Routes:
//   POST   /projects/{name}  → create project
//   GET    /projects         → list projects
//   DELETE /projects/{name}  → delete project
func ProjectsHandler(store *storage.Store, log logging.Logger) http.Handler
```

---

## Aggiornamento `internal/config`

Aggiunto il campo `KeysDir` a `Config` con la relativa variabile d'ambiente:

```go
KeysDir string  // EKVS_KEYS_DIR, default: "./data/.keys"
```

Il default `"./data/.keys"` è coerente con `StoragePath` (`"./data"`) e con il layout
`{storage_root}/.keys/` definito in `server_auth`. La modifica è **retrocompatibile**:
nessun package esistente usa `Config` direttamente oltre a `cmd/server/main.go`.

---

## Dipendenze dai milestone precedenti

| Simbolo | Provenienza |
|---------|-------------|
| `storage.Store` | `ekvs/internal/storage` (server_storage) |
| `storage.ErrProjectNotFound` | `ekvs/internal/storage` |
| `storage.ErrProjectAlreadyExists` | `ekvs/internal/storage` |
| `storage.ErrInvalidName` | `ekvs/internal/storage` |
| `auth.AuthMiddleware` | `ekvs/internal/auth` (server_auth) |
| `auth.UserIDFromContext` | `ekvs/internal/auth` |
| `auth.NewKeyStore` | `ekvs/internal/auth` |
| `config.Load` | `ekvs/internal/config` (project_setup) |
| `logging.New` / `logging.Logger` | `ekvs/internal/logging` (project_setup) |



