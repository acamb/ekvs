# requirements.md — server_storage

## User Decisions

| Decision | Choice |
|---|---|
| Directory structure | `{storage_root}/{userID_sanitized}/{project_name}.json` |
| File format | JSON with metadata: `{"version": 1, "entries": {"KEY": "base64blob"}}` |
| Naming validation | `^[a-zA-Z0-9_\-\.]{1,128}$` — applied to both project names and secret keys |
| Concurrency model | Single-process; one `sync.RWMutex` **per project**; a meta `sync.Mutex` protects the lock map |
| Atomic writes | Write to temp file in same directory, then `os.Rename` |
| userID sanitisation | Replace every character outside `[a-zA-Z0-9_\-]` with `_` (deterministic, no validation error) |
| JSON versioning | `"version"` field stored as `1`; reads that encounter an unknown version return a wrapped `fmt.Errorf` (no migration logic in this milestone) |

---

## Scope

### In scope
- Struct `Store` with a constructor `New(dir string) (*Store, error)`.
- Project CRUD: `CreateProject`, `DeleteProject`, `ListProjects`.
- Secret CRUD: `SetSecret`, `GetSecret`, `DeleteSecret`, `ListSecrets`.
- Package-level error sentinels: `ErrProjectNotFound`, `ErrProjectAlreadyExists`, `ErrKeyNotFound`, `ErrInvalidName`.
- Internal helpers: `validateName`, `sanitizeID`, `projectPath`, atomic file write.
- Unit tests (table-driven) with ≥ 90 % statement coverage.

### Out of scope
- HTTP handlers or middleware (those live in `server_projects_api` / `server_secrets_api`).
- User registration / `authorized_keys` management (those live in `server_auth`).
- Encryption or decryption of values (the store treats values as opaque strings).
- Schema migration between `"version"` numbers.
- Multi-process file locking.

---

## Package Path

```
internal/storage
```

Import path: `ekvs/internal/storage`

---

## Exported API

```go
package storage

import "errors"

// Error sentinels.
var (
    ErrProjectNotFound    = errors.New("project not found")
    ErrProjectAlreadyExists = errors.New("project already exists")
    ErrKeyNotFound        = errors.New("key not found")
    ErrInvalidName        = errors.New("invalid name")
)

// Store is a file-backed storage engine for encrypted key-value pairs.
// It is safe for concurrent use within a single process.
// Different projects can be read/written in parallel; access to the
// same project is serialised via a per-project RWMutex.
type Store struct { /* unexported */ }

// New creates (or opens) a Store rooted at dir.
// dir and all parent directories are created if they do not exist.
func New(dir string) (*Store, error)

// CreateProject creates a new empty project file for userID.
// Returns ErrInvalidName if project does not match the naming regex.
// Returns ErrProjectAlreadyExists if the project already exists.
func (s *Store) CreateProject(userID, project string) error

// DeleteProject removes the project file and all its secrets.
// Returns ErrProjectNotFound if the project does not exist.
func (s *Store) DeleteProject(userID, project string) error

// ListProjects returns all project names for userID, sorted alphabetically.
// Returns an empty (non-nil) slice if the user has no projects.
func (s *Store) ListProjects(userID string) ([]string, error)

// SetSecret creates or overwrites key with value in project.
// Returns ErrProjectNotFound if the project does not exist.
// Returns ErrInvalidName if key does not match the naming regex.
func (s *Store) SetSecret(userID, project, key, value string) error

// GetSecret returns the value stored under key in project.
// Returns ErrProjectNotFound if the project does not exist.
// Returns ErrKeyNotFound if the key does not exist.
func (s *Store) GetSecret(userID, project, key string) (string, error)

// DeleteSecret removes key from project.
// Returns ErrProjectNotFound if the project does not exist.
// Returns ErrKeyNotFound if the key does not exist.
func (s *Store) DeleteSecret(userID, project, key string) error

// ListSecrets returns all key names in project, sorted alphabetically.
// Returns an empty (non-nil) slice if the project has no secrets.
// Returns ErrProjectNotFound if the project does not exist.
func (s *Store) ListSecrets(userID, project string) ([]string, error)
```

---

## File Format

Each project is stored as a single JSON file:

```json
{
  "version": 1,
  "entries": {
    "DB_PASSWORD": "base64encodedciphertextblob==",
    "API_KEY":     "anotherbase64blob=="
  }
}
```

| Field | Type | Notes |
|---|---|---|
| `version` | `int` | Always `1` in this milestone. Unknown values return an error on read. |
| `entries` | `map[string]string` | Keys are plaintext identifiers; values are opaque encrypted blobs (base64 strings provided by the client). |

---

## Directory Layout

```
{storage_root}/
  {sanitized_fingerprint_A}/
    projectA.json
    projectB.json
  {sanitized_fingerprint_B}/
    projectC.json
```

`sanitizeID` transforms a `userID` (typically an SSH fingerprint like `SHA256:abc+def/ghi=`) into a filesystem-safe directory name by replacing every character outside `[a-zA-Z0-9_\-]` with `_`:

```
SHA256:abc+def/ghi=  →  SHA256_abc_def_ghi_
```

---

## Naming Validation

Regex: `^[a-zA-Z0-9_\-\.]{1,128}$`

Applied to:
- `project` argument in all methods.
- `key` argument in secret methods.

**Not** applied to `userID` (sanitised silently instead).

---

## Concurrency

The `Store` uses two levels of locking:

| Level | Field | Type | Protects |
|---|---|---|---|
| Meta | `Store.mu` | `sync.Mutex` | The `locks` map (insertion/lookup) |
| Per-project | `locks[lockKey]` | `*sync.RWMutex` | All filesystem I/O for that project |

`lockKey` is `sanitizeID(userID) + "/" + project`.

Locking protocol for any operation on `(userID, project)`:
1. Acquire `Store.mu`.
2. Look up (or create) `locks[lockKey]`.
3. Release `Store.mu`.
4. Acquire a **read lock** (reads) or **write lock** (writes) on `locks[lockKey]`.
5. Perform the filesystem operation.
6. Release the per-project lock.

This allows operations on **different projects to proceed in parallel** while
serialising concurrent access to the **same project**.

---

## Atomic Write Protocol

1. Marshal the updated `projectFile` struct to JSON.
2. Write to a temp file in the same directory (using `os.CreateTemp`).
3. Call `os.Rename(tempPath, targetPath)` — atomic on POSIX and Windows (NT).
4. On any error after step 2, remove the temp file.

---

## Error Sentinels

| Sentinel | Returned by | Condition |
|---|---|---|
| `ErrInvalidName` | `CreateProject`, `SetSecret` | Name fails regex validation |
| `ErrProjectAlreadyExists` | `CreateProject` | Project file already exists |
| `ErrProjectNotFound` | `DeleteProject`, `SetSecret`, `GetSecret`, `DeleteSecret`, `ListSecrets` | Project file does not exist |
| `ErrKeyNotFound` | `GetSecret`, `DeleteSecret` | Key absent from `entries` map |

---

## Dependencies

No new module dependencies. Standard library only:
`encoding/json`, `os`, `path/filepath`, `regexp`, `sort`, `sync`.

---

## Testing Requirements

- Framework: standard `testing` package; table-driven tests.
- Each test uses `t.TempDir()` for full isolation.
- Persistence test: write via one `Store` instance, read back via a new `Store` instance on the same directory.
- Concurrency test: multiple goroutines calling `SetSecret` / `GetSecret` concurrently on the same store — must not race (verified with `-race`).
- Coverage target: **≥ 90 % statement coverage** for `ekvs/internal/storage`.




