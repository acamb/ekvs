# validation.md — tui_projects

## Unit Tests

### `internal/tui/client` package

| Test name | What it verifies |
|-----------|-----------------|
| `TestListProjects_Success` | `GET /projects` returns `200` with body `{"projects":["a","b"]}` → `ListProjects()` returns `["a","b"], nil` |
| `TestListProjects_Empty` | Server returns `{"projects":[]}` → returns empty slice, no error |
| `TestListProjects_Unauthorized` | Server returns `401` → `ErrUnauthorized` |
| `TestCreateProject_Success` | `POST /projects/foo` returns `201` → no error |
| `TestCreateProject_Conflict` | Server returns `409` → `ErrConflict` |
| `TestCreateProject_BadRequest` | Server returns `400` → `ErrServer` |
| `TestDeleteProject_Success` | `DELETE /projects/foo` returns `204` → no error |
| `TestDeleteProject_NotFound` | Server returns `404` → `ErrNotFound` |
| `TestClient_NetworkError` | Server closed before request → error returned (not nil) |
| `TestClient_SignsRequests` | All three auth headers (`X-Timestamp`, `X-Fingerprint`, `X-Signature`) are present and non-empty in every outgoing request |

### `internal/tui/projects` package

| Test name | What it verifies |
|-----------|-----------------|
| `TestProjectsModel_FetchUpdates` | Sending `FetchedMsg{["a","b","c"]}` sets `projects` field and clears `loading` |
| `TestProjectsModel_ErrDisplayed` | Sending `ErrMsg{err}` sets `err` field; `View()` output contains error text |
| `TestProjectsModel_CursorDown` | Pressing `down` moves cursor from 0 → 1; wraps at end of page |
| `TestProjectsModel_CursorUp` | Pressing `up` wraps from 0 → last item on page |
| `TestProjectsModel_Pagination` | With 15 projects and pageSize=10: initial page=0 shows items 0–9; pressing `right` shows items 10–14 |
| `TestProjectsModel_PaginationBoundary` | `left` on page 0 stays on page 0; `right` on last page stays on last page |
| `TestProjectsModel_CreateMode` | Pressing `n` switches to `modeCreate`; typing characters accumulates in `input`; `backspace` removes last character |
| `TestProjectsModel_CreateSubmit` | In `modeCreate`, pressing `enter` with valid name emits the create command |
| `TestProjectsModel_CreateValidation` | Pressing `enter` with empty name does not emit command; model stays in `modeCreate` with error message |
| `TestProjectsModel_CreateCancel` | Pressing `esc` in `modeCreate` returns to `modeList` |
| `TestProjectsModel_DeleteConfirm` | Pressing `d` switches to `modeDelete`; pressing `y` emits delete command |
| `TestProjectsModel_DeleteCancel` | Pressing `n` or `esc` in `modeDelete` returns to `modeList` without emitting command |
| `TestProjectsModel_BackOnEsc` | Pressing `esc` in `modeList` returns `BackMsg{}` |
| `TestProjectsModel_LoadingState` | `loading = true` before fetch; cleared by `FetchedMsg` or `ErrMsg` |

---

## Manual Validation (step-by-step)

### Prerequisites
- Server running and reachable at the URL in the active profile.
- The user's public key is registered in the server's `.keys/` directory.
- At least one project may or may not exist in advance.

---

### Scenario 1 — Auto-auth on first navigation to Projects

1. Launch TUI with a valid profile; do **not** authenticate manually.
2. From the main menu, select **Projects** and press `Enter`.
3. **Expected**: the auth screen appears (passphrase prompt if key is encrypted,
   or immediate transition if key is unencrypted).
4. Complete authentication.
5. **Expected**: the Projects screen appears immediately with the project list
   fetched from the server (may be empty).

---

### Scenario 2 — Browse paginated project list

1. Pre-create more than 10 projects (e.g. via CLI or curl).
2. Navigate to Projects.
3. **Expected**: first page shows 10 items; status bar shows "Page 1 / N".
4. Press `→` (right arrow).
5. **Expected**: next page displayed, cursor resets to top.
6. Press `←` (left arrow).
7. **Expected**: back to first page.

---

### Scenario 3 — Create a new project

1. Navigate to Projects.
2. Press `n`.
3. **Expected**: inline input row appears at the bottom: `New project name: _`.
4. Type `my-test-project` and press `Enter`.
5. **Expected**: input row disappears, list refreshes and includes
   `my-test-project`.
6. Press `n` again, type an existing project name, press `Enter`.
7. **Expected**: error line appears: `conflict: project already exists` (or
   similar); list unchanged.
8. Press `n`, leave input empty, press `Enter`.
9. **Expected**: inline error "name cannot be empty"; stays in create mode.
10. Press `Esc`.
11. **Expected**: create mode cancelled, back to list.

---

### Scenario 4 — Delete a project

1. Navigate to Projects; at least one project in the list.
2. Move cursor to the target project and press `d`.
3. **Expected**: inline confirm row: `Delete 'my-test-project'? [y/n]`.
4. Press `n`.
5. **Expected**: confirm row disappears; list unchanged.
6. Press `d` again, then press `y`.
7. **Expected**: project removed from list; list refreshes.

---

### Scenario 5 — Return to main menu

1. Navigate to Projects.
2. Press `Esc` (in list mode).
3. **Expected**: main menu displayed; session remains authenticated.

---

### Scenario 6 — Network / server error

1. Stop the server process.
2. Navigate to Projects (already authenticated).
3. **Expected**: "Loading…" briefly, then an error line: `failed to fetch
   projects: …` (or similar).
4. Pressing any navigation key should clear the error and allow retry or `Esc`.

---

## Confirming "Authenticate" removal

1. Launch TUI.
2. Inspect the main menu.
3. **Expected**: menu contains **Projects**, **Secrets**, **Settings**, **Quit**
   only — no "Authenticate" entry.

