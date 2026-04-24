# requirements.md — server_secrets_api

## User Decisions

| Decisione | Scelta |
|-----------|--------|
| Router | `net/http` standard library + `http.ServeMux` (nessuna dipendenza esterna) |
| Versione Go | Go 1.25 (da `go.mod`); routing con method nel pattern disponibile nativamente da Go 1.22 |
| Base path | `/projects/{name}` — tutti gli endpoint sono sotto questo prefisso |
| Autenticazione | `auth.AuthMiddleware` avvolge il router globale; `userID` estratto da contesto via `auth.UserIDFromContext` |
| Formato risposta | JSON (`Content-Type: application/json`) per tutte le risposte, sia successo che errore |
| Corpo errori | `{"error": "<messaggio>"}` — coerente con feature precedenti |
| Validazione chiave | Delegata a `storage.SetSecret` → `storage.ErrInvalidName` → `400 Bad Request` |
| Progetto non trovato | Qualsiasi operazione su progetto inesistente → `404 Not Found` |
| Chiave non trovata | `GetSecret` / `DeleteSecret` su chiave inesistente → `404 Not Found` |
| Helper condivisi | `writeJSON` e `writeError` spostati da `projects.go` in `internal/server/helpers.go` |
| Package path | `internal/server` |
| Logging | `internal/logging.Logger` iniettato in `SecretsHandler`; errori interni (`500`) loggati con `.Error()`; operazioni valide con `.Debug()` |
| Mounting | `SecretsHandler` montato sullo stesso `http.ServeMux` globale di `ProjectsHandler` in `cmd/server/main.go`, avvolto dallo stesso `auth.AuthMiddleware` |
| `GET /projects/{name}` | Restituisce metadati del progetto (nome) + lista delle chiavi dei segreti presenti nel progetto, verificando l'esistenza del progetto tramite `store.ListSecrets` |
| Valori dei segreti | Blob opachi (stringhe, tipicamente base64 cifrato lato client); il server li tratta come stringhe opache senza validazione del contenuto |
| Test strategy | Storage reale su disco (`t.TempDir()`); table-driven tests; copertura ≥ 90% degli statement |

---

## Scope

### In scope
- Refactor: spostare `writeJSON` e `writeError` da `projects.go` a `helpers.go` nel package `internal/server`.
- `SecretsHandler(store *storage.Store, log logging.Logger) http.Handler` — restituisce un `http.Handler` che espone le rotte per la gestione dei segreti.
- 5 endpoint REST (vedi sezione API).
- Traduzione degli errori di `internal/storage` in codici HTTP appropriati.
- Unit tests (table-driven) con copertura ≥ 90% degli statement, usando storage reale su `t.TempDir()`.
- Aggiornamento di `cmd/server/main.go` per cablare `SecretsHandler` sullo stesso mux globale.

### Out of scope
- Crittografia / decrittografia lato server (i valori arrivano già cifrati).
- Validazione del contenuto dei valori segreti.
- Gestione delle chiavi SSH — già implementata in `server_auth`.
- TLS / rate limiting / paginazione.
- Endpoint di gestione dei progetti — già implementati in `server_projects_api`.

---

## Package Path

```
internal/server
```

Import path: `ekvs/internal/server`

---

## Refactor preliminare

Prima di aggiungere `secrets.go`, gli helper `writeJSON` e `writeError` vanno spostati da `projects.go` in un nuovo file `helpers.go` nello stesso package. Nessuna firma cambia; è solo una riorganizzazione interna.

```go
// internal/server/helpers.go
package server

import (
    "encoding/json"
    "net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) { ... }
func writeError(w http.ResponseWriter, status int, msg string) { ... }
```

---

## API

### `GET /projects/{name}`
Restituisce i metadati del progetto e la lista delle chiavi dei segreti presenti.

**Request**: nessun corpo.

**Response successo**: `200 OK`
```json
{
  "name": "my-project",
  "keys": ["api_key", "db_password"]
}
```
Lista `keys` ordinata alfabeticamente (delegata a `storage.ListSecrets`). Se il progetto non ha segreti, `keys` è `[]`.

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Progetto non trovato (`storage.ErrProjectNotFound`) | `404 Not Found` |
| Errore interno | `500 Internal Server Error` |

---

### `PUT /projects/{name}/secrets/{key}`
Crea o sovrascrive un segreto nel progetto.

**Request body** (`Content-Type: application/json`):
```json
{"value": "<blob cifrato, stringa opaca>"}
```

**Response successo**: `200 OK`
```json
{"key": "db_password"}
```

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Body JSON mancante o malformato | `400 Bad Request` |
| Campo `value` assente o vuoto | `400 Bad Request` |
| Nome chiave non valido (`storage.ErrInvalidName`) | `400 Bad Request` |
| Progetto non trovato (`storage.ErrProjectNotFound`) | `404 Not Found` |
| Errore interno | `500 Internal Server Error` |

---

### `GET /projects/{name}/secrets/{key}`
Legge il valore di un segreto.

**Request**: nessun corpo.

**Response successo**: `200 OK`
```json
{"key": "db_password", "value": "<blob cifrato>"}
```

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Progetto non trovato (`storage.ErrProjectNotFound`) | `404 Not Found` |
| Chiave non trovata (`storage.ErrKeyNotFound`) | `404 Not Found` |
| Errore interno | `500 Internal Server Error` |

---

### `DELETE /projects/{name}/secrets/{key}`
Elimina un segreto dal progetto.

**Request**: nessun corpo.

**Response successo**: `204 No Content` (nessun corpo).

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Progetto non trovato (`storage.ErrProjectNotFound`) | `404 Not Found` |
| Chiave non trovata (`storage.ErrKeyNotFound`) | `404 Not Found` |
| Errore interno | `500 Internal Server Error` |

---

### `GET /projects/{name}/secrets`
Elenca tutti i segreti del progetto, restituendo sia le chiavi che i relativi valori.

**Request**: nessun corpo.

**Response successo**: `200 OK`
```json
{
  "secrets": [
    {"key": "api_key", "value": "<blob cifrato>"},
    {"key": "db_password", "value": "<blob cifrato>"}
  ]
}
```
Lista ordinata alfabeticamente per chiave (ottenuta chiamando `store.ListSecrets` per le chiavi ordinate, poi `store.GetSecret` per ciascuna). Se il progetto non ha segreti, `secrets` è `[]`.

**Response errori**:
| Condizione | Codice |
|------------|--------|
| Progetto non trovato (`storage.ErrProjectNotFound`) | `404 Not Found` |
| Errore interno | `500 Internal Server Error` |

---

## Exported API

```go
package server

import (
    "net/http"

    "ekvs/internal/logging"
    "ekvs/internal/storage"
)

// SecretsHandler returns an http.Handler that exposes secret management
// endpoints for a given project. It must be wrapped by auth.AuthMiddleware
// before mounting.
//
// Routes:
//
//  GET    /projects/{name}             → get project info + list keys
//  PUT    /projects/{name}/secrets/{key} → set secret
//  GET    /projects/{name}/secrets/{key} → get secret
//  DELETE /projects/{name}/secrets/{key} → delete secret
//  GET    /projects/{name}/secrets      → list secret keys
func SecretsHandler(store *storage.Store, log logging.Logger) http.Handler
```

---

## Mounting in `cmd/server/main.go`

`SecretsHandler` viene aggiunto allo stesso `http.ServeMux` globale usato da `ProjectsHandler`, prima di avvolgerlo con `auth.AuthMiddleware`:

```go
mux := http.NewServeMux()
mux.Handle("/projects/", server.ProjectsHandler(store, log))
mux.Handle("/projects/", server.SecretsHandler(store, log))  // rotte più specifiche hanno precedenza
handler := auth.AuthMiddleware(keystore, log)(mux)
```

> **Nota**: Go 1.22+ risolve le rotte per specificità decrescente del pattern. Le rotte `/projects/{name}/secrets/{key}` e `/projects/{name}/secrets` sono più specifiche di `/projects/{name}` e verranno correttamente disambiguate. In pratica conviene registrare le rotte con il metodo HTTP nel pattern (es. `"GET /projects/{name}"`) direttamente nel mux restituito da `SecretsHandler`, e montare l'handler su `/` del mux globale (o passare il mux globale direttamente a entrambe le factory).

