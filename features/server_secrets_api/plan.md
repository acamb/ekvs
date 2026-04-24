# plan.md — server_secrets_api

Lista ordinata dei task da eseguire per implementare la feature. Ogni task è atomico e verificabile singolarmente.

---

## Task

### 1. Refactor: estrarre `helpers.go`
Spostare le funzioni `writeJSON` e `writeError` da `internal/server/projects.go` al nuovo file `internal/server/helpers.go`. Aggiornare `projects.go` rimuovendo le funzioni duplicate. Verificare che la compilazione e i test esistenti passino senza modifiche.

---

### 2. Creare `internal/server/secrets.go` — skeleton e routing
Creare il file `internal/server/secrets.go` con la funzione `SecretsHandler(store *storage.Store, log logging.Logger) http.Handler`. Registrare nel mux interno le cinque rotte (con metodo HTTP nel pattern) senza logica di business: tutti gli handler restituiscono `501 Not Implemented` come placeholder. Verificare la compilazione.

---

### 3. Implementare `GET /projects/{name}`
Implementare l'handler che estrae `userID` dal contesto, chiama `store.ListSecrets(userID, name)` e risponde con `{"name": name, "keys": [...]}`. Tradurre `ErrProjectNotFound` → `404`. Lista `keys` mai `null`: usare `[]string{}` come default se vuota.

---

### 4. Implementare `GET /projects/{name}/secrets`
Implementare l'handler che:
1. Chiama `store.ListSecrets(userID, name)` per ottenere le chiavi ordinate alfabeticamente.
2. Per ogni chiave chiama `store.GetSecret(userID, name, key)` per recuperare il valore.
3. Risponde con `{"secrets": [{"key": "...", "value": "..."}, ...]}`.
Tradurre `ErrProjectNotFound` → `404`. Lista `secrets` mai `null`: usare `[]` come default se vuota.

---

### 5. Implementare `PUT /projects/{name}/secrets/{key}`
Implementare l'handler che:
1. Decodifica il body JSON aspettandosi `{"value": "..."}`.
2. Valida che `value` non sia vuoto → `400`.
3. Chiama `store.SetSecret(userID, name, key, value)`.
4. Traduce `ErrInvalidName` → `400`, `ErrProjectNotFound` → `404`.
5. Risponde `200 OK` con `{"key": key}`.

---

### 6. Implementare `GET /projects/{name}/secrets/{key}`
Implementare l'handler che chiama `store.GetSecret(userID, name, key)` e risponde con `{"key": key, "value": value}`. Tradurre `ErrProjectNotFound` e `ErrKeyNotFound` → `404`.

---

### 7. Implementare `DELETE /projects/{name}/secrets/{key}`
Implementare l'handler che chiama `store.DeleteSecret(userID, name, key)` e risponde `204 No Content`. Tradurre `ErrProjectNotFound` e `ErrKeyNotFound` → `404`.

---

### 8. Scrivere unit test per `SecretsHandler`
Aggiungere casi table-driven in `internal/server/server_test.go` (o in un nuovo file `secrets_test.go` nello stesso package) che coprano per ogni endpoint:
- Path happy: risposta corretta con body e status attesi.
- Progetto non trovato → `404`.
- Chiave non trovata → `404` (per GET/DELETE secret).
- Nome chiave invalido → `400` (per PUT).
- Body malformato / `value` vuoto → `400` (per PUT).
- `userID` mancante nel contesto → `500`.

Usare storage reale su `t.TempDir()`. Copertura target ≥ 90% degli statement del package.

---

### 9. Aggiornare `cmd/server/main.go`
Registrare `SecretsHandler` sullo stesso `http.ServeMux` globale in `cmd/server/main.go`, prima di applicare `auth.AuthMiddleware`. Verificare che la compilazione dell'intero modulo sia pulita (`go build ./...`).

---

### 10. Verifica finale
Eseguire `make test` (o `go test ./...`) e controllare che tutti i test passino e la copertura del package `internal/server` sia ≥ 90%. Eseguire `go vet ./...` e assicurarsi che non ci siano warning.

