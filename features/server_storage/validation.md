# validation.md — server_storage

## How to Run Tests

```bash
# Unit tests with race detector and coverage:
go test -race -count=1 -cover ./internal/storage/...

# Detailed per-function coverage:
go test -count=1 -coverprofile=coverage.out ./internal/storage/...
go tool cover -func=coverage.out

# Full suite (must not regress):
make test
```

**"Passing" definition:** all tests exit `PASS`, no data races, statement coverage
≥ 90 % for `ekvs/internal/storage`, and `make test` remains fully green.

---

## Manual Checklist

- [ ] `internal/storage/errors.go` exports exactly `ErrProjectNotFound`, `ErrProjectAlreadyExists`, `ErrKeyNotFound`, `ErrInvalidName`, `ErrUnknownVersion`.
- [ ] `go build ./internal/storage/...` produces no errors.
- [ ] `go vet ./internal/storage/...` produces no diagnostics.
- [ ] `go mod tidy` leaves `go.mod` / `go.sum` unchanged.
- [ ] Project files are written as valid JSON with `"version": 1`.
- [ ] Project JSON files are created with mode `0600`; user directories with mode `0700`.
- [ ] Two consecutive `Encrypt` calls on the same key return the new value (upsert semantics).
- [ ] `ListProjects` and `ListSecrets` both return results in **alphabetical order**.
- [ ] `ListProjects` returns an empty non-nil slice (no error) when the user directory does not exist.
- [ ] A project created by `Store` A is visible when opening a new `Store` B on the same directory (persistence).
- [ ] Running the concurrency test with `-race` reports **no data races**.
- [ ] Invalid names (`""`, `"a/b"`, `"x"*129`) return `ErrInvalidName`.
- [ ] `userID` containing SSH fingerprint characters (`:`, `+`, `/`, `=`) does **not** return an error; the sanitised directory is created instead.
- [ ] A project file with `"version": 99` causes `ErrUnknownVersion` (inspectable via `errors.Is`).
- [ ] After `DeleteProject` succeeds, the per-project lock entry is removed from the internal map (verify via a subsequent `CreateProject` on the same name acquiring a fresh lock).

---

## Test-Case Matrix

### `New`

| # | Input | Expected |
|---|---|---|
| N-1 | Fresh (non-existent) directory | Directory created, no error |
| N-2 | Pre-existing directory | No error, existing files untouched |

### `CreateProject`

| # | Input | Expected |
|---|---|---|
| CP-1 | Valid userID + valid project name | Project file created, no error |
| CP-2 | Same project created twice | `ErrProjectAlreadyExists` |
| CP-3 | Invalid project name (empty) | `ErrInvalidName` |
| CP-4 | Invalid project name (contains `/`) | `ErrInvalidName` |
| CP-5 | Invalid project name (> 128 chars) | `ErrInvalidName` |

### `DeleteProject`

| # | Input | Expected |
|---|---|---|
| DP-1 | Existing project | File removed, no error |
| DP-2 | Non-existent project | `ErrProjectNotFound` |

### `ListProjects`

 #  Input  Expected 
---------
 LP-1  User with no projects / user directory does not exist yet  Empty slice (non-nil), no error (`os.ErrNotExist` absorbed) 
 LP-2  User with one project  `["proj"]`, no error 
 LP-3  User with multiple projects  Names sorted alphabetically 

### `SetSecret`

| # | Input | Expected |
|---|---|---|
| SS-1 | New key in existing project | Entry added, no error |
| SS-2 | Overwrite existing key | Entry updated, no error |
| SS-3 | Project does not exist | `ErrProjectNotFound` |
| SS-4 | Invalid key name | `ErrInvalidName` |

### `GetSecret`

| # | Input | Expected |
|---|---|---|
| GS-1 | Existing key | Returns stored value, no error |
| GS-2 | Non-existent key | `ErrKeyNotFound` |
| GS-3 | Project does not exist | `ErrProjectNotFound` |

### `DeleteSecret`

| # | Input | Expected |
|---|---|---|
| DS-1 | Existing key | Entry removed, no error |
| DS-2 | Non-existent key | `ErrKeyNotFound` |
| DS-3 | Project does not exist | `ErrProjectNotFound` |

### `ListSecrets`

| # | Input | Expected |
|---|---|---|
| LS-1 | Empty project | Empty slice, no error |
| LS-2 | Project with multiple keys | Keys sorted alphabetically |
| LS-3 | Project does not exist | `ErrProjectNotFound` |

### `Persistence`

 #  Scenario  Expected 
---------
 P-1  Create project + set secret via Store A; open Store B on same dir; get secret  Value equal to original 
 P-2  Delete project via Store A; open Store B; list projects  Project not present 

### `Version handling`

 #  Scenario  Expected 
---------
 V-1  Manually write a project file with `"version": 99`; call `GetSecret`  `ErrUnknownVersion` returned; no panic 

### Concurrency

| # | Scenario | Expected |
|---|---|---|
 C-1  N goroutines each set a unique key in the **same** project  All keys present after join; `-race` reports no races 
 C-2  N goroutines each operate on a **different** project simultaneously  All complete without blocking each other; `-race` reports no races 
 C-3  Concurrent reads while one goroutine writes to the **same** project  Each read returns either the value present **before** or **after** the write (both are valid); no partially-written data is ever observed; `-race` reports no races 
 C-4  Delete a project while N goroutines attempt concurrent reads/writes on it  Operations on the deleted project return `ErrProjectNotFound`; lock map does not retain the entry after deletion; `-race` reports no races 


---


