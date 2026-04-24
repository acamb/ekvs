# plan.md — server_projects_api

## Ordered Task List

1. **Aggiornare `internal/config`**
   - Aggiungere campo `KeysDir string` a `Config`.
   - Aggiungere costante `defaultKeysDir = "./data/.keys"`.
   - Aggiungere `KeysDir: envOr("EKVS_KEYS_DIR", defaultKeysDir)` in `Load()`.
   - Aggiornare `config_test.go`: verificare che `EKVS_KEYS_DIR` venga letto correttamente e che il default sia applicato in assenza della variabile.

2. **Creare lo skeleton del package `internal/server`**
   Creare:
   - `internal/server/projects.go` — `ProjectsHandler` e helpers interni
   - `internal/server/server_test.go` — tutti gli unit test

3. **Implementare gli helper interni (`projects.go`)**
   - `writeJSON(w http.ResponseWriter, status int, v any)` — imposta `Content-Type: application/json`, scrive lo status code, serializza con `json.NewEncoder`.
   - `writeError(w http.ResponseWriter, status int, msg string)` — chiama `writeJSON` con `map[string]string{"error": msg}`.

4. **Implementare `ProjectsHandler` (`projects.go`)**
   Firma: `func ProjectsHandler(store *storage.Store, log logging.Logger) http.Handler`

   Usa `http.NewServeMux()` locale con pattern Go 1.22+.

   **`POST /projects/{name}`** (`handleCreateProject`):
   1. Estrarre `userID` con `auth.UserIDFromContext`; se assente → `log.Error(...)` + `500`.
   2. Estrarre `name` da `r.PathValue("name")`.
   3. `store.CreateProject(userID, name)`:
      - `errors.Is(err, storage.ErrInvalidName)` → `400`.
      - `errors.Is(err, storage.ErrProjectAlreadyExists)` → `409`.
      - altro errore → `log.Error(...)` + `500`.
   4. Successo → `log.Debug(...)` + `201` con `{"name": name}`.

   **`GET /projects`** (`handleListProjects`):
   1. Estrarre `userID`; se assente → `log.Error(...)` + `500`.
   2. `store.ListProjects(userID)`:
      - errore → `log.Error(...)` + `500`.
   3. Successo → `log.Debug(...)` + `200` con `{"projects": [...]}`.
   > **Nota**: `storage.ListProjects` restituisce già `[]string{}` (mai nil) quando
   > l'utente non ha progetti o la sua directory non esiste ancora. Il handler non
   > deve normalizzare.

   **`DELETE /projects/{name}`** (`handleDeleteProject`):
   1. Estrarre `userID`; se assente → `log.Error(...)` + `500`.
   2. Estrarre `name` da `r.PathValue("name")`.
   3. `store.DeleteProject(userID, name)`:
      - `errors.Is(err, storage.ErrProjectNotFound)` → `404`.
      - altro errore → `log.Error(...)` + `500`.
   4. Successo → `log.Debug(...)` + `204` senza corpo.

   > **Nota implementativa**: `storage.DeleteProject` non chiama `validateName`, quindi
   > un nome sintatticamente invalido non produce `ErrInvalidName` ma `ErrProjectNotFound`
   > (il file non può esistere su disco). Il handler **non pre-valida** il nome: un nome
   > invalido restituisce `404`.

   Registrazione rotte:
   ```go
   mux.HandleFunc("POST /projects/{name}", handleCreateProject)
   mux.HandleFunc("GET /projects", handleListProjects)
   mux.HandleFunc("DELETE /projects/{name}", handleDeleteProject)
   ```

5. **Cablare il server in `cmd/server/main.go`**
   ```go
   cfg, _ := config.Load()
   log    := logging.New(cfg.LogLevel)
   store, _ := storage.New(cfg.StoragePath)
   ks, _    := auth.NewKeyStore(cfg.KeysDir)
   handler  := server.ProjectsHandler(store, log)
   authed   := auth.AuthMiddleware(ks, 30*time.Second, handler)
   http.ListenAndServe(cfg.ServerAddr, authed)
   ```
   Gestire gli errori fatali con `log.Error(...)` + `os.Exit(1)`.

6. **Scrivere gli unit test (`server_test.go`)**
   Tutti i test usano storage reale su `t.TempDir()` + `httptest.NewRecorder` / `httptest.NewRequest`.
   Il contesto con userID viene iniettato manualmente con `auth.UserIDFromContext` (o
   direttamente con `context.WithValue` usando la chiave interna dove necessario per
   simulare contesto mancante).

   Table-driven per ogni handler:

   **`handleCreateProject`**:
   - Richiesta valida → `201`, corpo `{"name":"alpha"}`.
   - Nome non valido (es. `"!!bad!!"`) → `400`.
     > Non usare `"../bad"` come esempio: Go normalizza `/projects/../bad` → `/bad`
     > prima che raggiunga il mux, producendo un `404` anziché un `400`.
     > Usare caratteri invalidi senza `/` (es. `!!`, spazi, `@`).
   - Progetto già esistente (crea due volte lo stesso) → `409`.

   **`handleListProjects`**:
   - Utente senza progetti → `200`, `{"projects":[]}`.
   - Utente con più progetti → `200`, lista ordinata.

   **`handleDeleteProject`**:
   - Progetto esistente → `204`.
   - Progetto non trovato → `404`.
   - Nome invalido (es. `"!!bad!!"`) → `404` (il file non può esistere).

   **Test end-to-end leggero** (storage reale, un unico test):
   - Crea progetto → lista (verifica presenza) → elimina → lista (verifica assenza).

   **Test isolamento tra utenti**:
   - UserID A crea `alpha`; userID B lista → lista vuota.

7. **Eseguire `go mod tidy` e validare**
   ```bash
   go mod tidy   # nessuna nuova dipendenza attesa
   make test     # tutti i package devono passare, incluso ekvs/internal/server
   ```
   Confermare copertura ≥ 90% su `internal/server` e nessuna regressione sugli altri package.




