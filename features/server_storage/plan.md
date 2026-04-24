# plan.md — server_storage

## Ordered Task List

1. **Create package skeleton**
   Create `internal/storage/` with:
   - `errors.go` — error sentinels
   - `storage.go` — `Store` struct, constructor, internal helpers
   - `projects.go` — project-level methods
   - `secrets.go` — secret-level methods
   - `storage_test.go` — all unit tests

2. **Define error sentinels (`errors.go`)**
   Declare:
   - `ErrProjectNotFound`
   - `ErrProjectAlreadyExists`
   - `ErrKeyNotFound`
   - `ErrInvalidName`

3. **Define internal types and helpers (`storage.go`)**
   - `projectFile` struct: `Version int`, `Entries map[string]string`.
   - `nameRE` compiled regex: `^[a-zA-Z0-9_\-\.]{1,128}$`.
   - `validateName(s string) error` — returns `ErrInvalidName` on mismatch.
   - `sanitizeID(userID string) string` — replaces `[^a-zA-Z0-9_\-]` with `_`.
   - `projectPath(root, userID, project string) string` — builds full path.
   - `lockKey(sanitizedID, project string) string` — returns `sanitizedID+"/"+project`.
   - `readProject(path string) (*projectFile, error)` — reads and unmarshals JSON; errors on unknown version.
   - `writeProject(path string, pf *projectFile) error` — atomic write (temp + rename).

4. **Implement `Store` constructor (`storage.go`)**
   `New(dir string) (*Store, error)`:
   - Call `os.MkdirAll(dir, 0700)`.
   - Initialise `locks map[string]*sync.RWMutex` and `mu sync.Mutex`.
   - Store `dir` in the struct.

   Internal helper `projectLock(userID, project string) *sync.RWMutex`:
   - Lock `s.mu`, look up (or lazily create) the `*sync.RWMutex` for `lockKey`, unlock `s.mu`, return it.

5. **Implement project methods (`projects.go`)**
   - `CreateProject` — validate name, acquire write lock for the project, check file does not exist, write empty `projectFile`.
   - `DeleteProject` — acquire write lock for the project, check file exists, `os.Remove`.
   - `ListProjects` — acquire `Store.mu` to get a snapshot of the user's dir, then `os.ReadDir`, filter `*.json`, strip extension, sort, return. (No per-project lock needed: listing only reads directory entries.)

6. **Implement secret methods (`secrets.go`)**
   - `SetSecret` — validate key name, acquire write lock for the project, read project, upsert entry, write back.
   - `GetSecret` — acquire read lock for the project, read project, lookup key, return value or `ErrKeyNotFound`.
   - `DeleteSecret` — acquire write lock for the project, read project, check key exists, delete entry, write back.
   - `ListSecrets` — acquire read lock for the project, read project, collect keys, sort, return.

7. **Write unit tests (`storage_test.go`)**
   Table-driven tests covering:
   - `New` with a fresh directory and a pre-existing directory.
   - `CreateProject` happy path, duplicate, invalid name.
   - `DeleteProject` happy path, not found.
   - `ListProjects` empty, single, multiple (sorted).
   - `SetSecret` create new, overwrite existing, project not found, invalid key.
   - `GetSecret` happy path, key not found, project not found.
   - `DeleteSecret` happy path, key not found, project not found.
   - `ListSecrets` empty, multiple (sorted), project not found.
   - Persistence: write via `Store` A, read back via `Store` B on the same dir.
   - Concurrency: N goroutines each calling `SetSecret` then `GetSecret`; verify no races.

8. **Run `go mod tidy` and validate**
   Run `go mod tidy` (no new deps), then `make test`; confirm `ok  ekvs/internal/storage`.



