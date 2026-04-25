# requirements.md — tui_projects

## Goal

Implement an interactive project management screen in the TUI client.
The user can browse, create, and delete projects owned by their authenticated
identity.  All API calls are signed using the session credentials produced by
the `tui_auth` flow.  If the session is not yet authenticated when the user
navigates to the Projects screen, the auth flow is triggered automatically;
on success the TUI returns to the Projects screen and fetches the list.

---

## Scope

### In scope
- `internal/tui/client` package: typed HTTP client that wraps `net/http`,
  handles request signing, JSON encoding/decoding, and maps HTTP error codes
  to typed Go errors.  Designed to be reused by `tui_secrets`.
- `internal/tui/projects` package: bubbletea model for the Projects screen.
- Removal of the temporary **"Authenticate"** menu item from `menu.go`; the
  menu now contains: Projects, Secrets, Settings, Quit.
- Integration of `screenProjects` into the root model lifecycle:
  - if `session.IsAuthenticated()` is false → trigger auth flow, then navigate
    to projects on `auth.AuthSuccessMsg`;
  - if already authenticated → navigate directly to projects.
- A scrollable, paginated project list (page size: configurable constant,
  default 10).
- **Create project** — inline text input row appended below the list when the
  user presses `n` (new); confirmed with `Enter`, cancelled with `Esc`.
- **Delete project** — inline confirmation row (`y/n`) shown below the
  selected item when the user presses `d` (delete); confirmed with `y`,
  cancelled with `n` or `Esc`.
- Status bar showing: current page / total pages, keyboard shortcuts.
- Error display: API errors are shown as a styled error line at the bottom of
  the screen; they are cleared on the next action.
- Unit tests for the `client` package (using an `httptest` server) and for the
  `projects` model (message-driven tests).

### Out of scope
- Encryption of secret values (deferred to `tui_encryption` / `tui_secrets`).
- Renaming or editing projects (not supported by the server API).
- SSH agent support (deferred to `ssh_agent_support`, Phase 6).
- Settings screen (future milestone).
- CLI client (separate milestones).

---

## Decisions

| # | Decision |
|---|----------|
| 1 | A new `internal/tui/client` package is created instead of adding methods to `session.Session`, to keep transport concerns separate and to make the package reusable for `tui_secrets`. |
| 2 | `client.Client` is constructed with the server base URL (from `profile.ServerURL`) and a `*session.Session` pointer; it calls `auth.SignRequest` internally before each request. |
| 3 | Create / delete interactions use **inline rows** (Option B) rather than full-screen overlays, to keep the project list visible and reduce context switching. |
| 4 | Pagination is implemented in the model: the model holds `page int` and a constant `pageSize = 10`; the full list is fetched once on screen entry and sliced client-side for display. |
| 5 | The Projects screen fetches the project list from the server immediately on entry by emitting a `fetchProjectsCmd`. The list is re-fetched after every successful create or delete. |
| 6 | If the session is not authenticated when "Projects" is selected from the main menu, `triggerAuthMsg` is sent with `returnTo = screenProjects`. On `auth.AuthSuccessMsg`, the root model navigates to `screenProjects` (same logic as the existing auth flow). |
| 7 | The temporary **"Authenticate"** menu entry is removed from `menu.go`. Authentication is now triggered implicitly by the Projects (and future Secrets) screen. |
| 8 | `client.Client` returns typed sentinel errors: `ErrUnauthorized`, `ErrNotFound`, `ErrConflict`, `ErrServer`. The projects model maps these to user-visible strings. |
| 9 | Project names are validated client-side (non-empty, no path separators) before sending to the server, giving instant feedback without a round-trip. |
| 10 | The projects model lives in a new sub-package `internal/tui/projects` (not under `root`), following the same layout as `internal/tui/auth`. |

---

## HTTP API contract (server side, already implemented)

| Method | Path | Success | Notes |
|--------|------|---------|-------|
| `GET` | `/projects` | `200 {"projects":["a","b"]}` | Returns all projects for the authenticated user |
| `POST` | `/projects/{name}` | `201` | Creates project; `409` if exists, `400` if invalid name |
| `DELETE` | `/projects/{name}` | `204` | Deletes project; `404` if not found |

All requests must carry `X-Timestamp`, `X-Fingerprint`, `X-Signature` headers
produced by `auth.SignRequest`.

---

## User Interaction Flow

```
Main Menu → [Enter on "Projects"]
  └─ session authenticated?
       ├─ YES → emit fetchProjectsCmd → show list
       └─ NO  → trigger auth flow
                └─ auth success → emit fetchProjectsCmd → show list

Projects screen (list mode):
  ↑/↓  k/j    move cursor
  →  / ←       change page
  n            enter "create" mode (inline input at bottom)
  d            enter "delete confirm" mode (inline prompt at bottom)
  Backspace/Esc exit to main menu

Create mode (inline):
  type name → Enter confirm → POST /projects/{name} → re-fetch list
                              Esc cancel

Delete mode (inline):
  y confirm → DELETE /projects/{name} → re-fetch list
  n / Esc cancel
```

---

## File layout

```
internal/tui/client/
    client.go          — Client struct, Do(), helper methods (ListProjects, CreateProject, DeleteProject)
    errors.go          — ErrUnauthorized, ErrNotFound, ErrConflict, ErrServer
    client_test.go     — tests against httptest.Server

internal/tui/projects/
    projects.go        — Model, Init, Update, View
    messages.go        — projectsFetchedMsg, projectsErrMsg
    projects_test.go   — unit tests

internal/tui/root/
    menu.go            — remove "authenticate" item; add "projects" enter handler
    root.go            — add screenProjects, projects model field, wiring
    messages.go        — add screenProjects to pendingScreen destinations (if needed)
```

