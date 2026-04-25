# plan.md — tui_projects

Ordered list of implementation tasks. Each task is self-contained and can be
reviewed independently.

---

## Task 1 — HTTP client package (`internal/tui/client`)

Create `internal/tui/client/errors.go`:
- Define sentinel errors: `ErrUnauthorized`, `ErrNotFound`, `ErrConflict`,
  `ErrServer` (wraps status code and body).

Create `internal/tui/client/client.go`:
- `Client` struct holding `baseURL string` and `sess *session.Session`.
- `New(baseURL string, sess *session.Session) *Client` constructor.
- Internal `do(method, path string, body io.Reader) (*http.Response, error)`
  method: calls `auth.SignRequest`, sets headers, executes the request.
- `ListProjects() ([]string, error)` — `GET /projects`, decodes
  `{"projects":[...]}`.
- `CreateProject(name string) error` — `POST /projects/{name}`.
- `DeleteProject(name string) error` — `DELETE /projects/{name}`.
- All methods map HTTP status codes to typed sentinel errors.

Create `internal/tui/client/client_test.go`:
- Table-driven tests using `net/http/httptest.NewServer`.
- Cover: successful list/create/delete, `ErrUnauthorized` (401),
  `ErrNotFound` (404), `ErrConflict` (409), network error.

---

## Task 2 — Projects bubbletea model (`internal/tui/projects`)

Create `internal/tui/projects/messages.go`:
- `FetchedMsg` carrying `[]string` (project names).
- `ErrMsg` carrying `error`.

Create `internal/tui/projects/projects.go`:
- `Model` struct fields:
  - `client *client.Client`
  - `theme theme.Theme`
  - `projects []string` (full list from server)
  - `cursor int` (index within current page)
  - `page int`
  - `pageSize int` (constant = 10)
  - `mode` enum: `modeList`, `modeCreate`, `modeDelete`
  - `input string` (for create mode)
  - `err error`
  - `loading bool`
- `New(c *client.Client, t theme.Theme) Model` constructor.
- `Init() tea.Cmd` — emits `fetchProjectsCmd`.
- `fetchProjectsCmd` — calls `client.ListProjects()` off the main goroutine;
  returns `FetchedMsg` or `ErrMsg`.
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
  - `FetchedMsg` → update list, reset cursor/page, `loading = false`.
  - `ErrMsg` → set `err`, `loading = false`.
  - `tea.KeyPressMsg` (list mode):
    - `up`/`k`, `down`/`j` → move cursor (wraps within page).
    - `right` / `left` → change page.
    - `n` → switch to `modeCreate`, clear input.
    - `d` → switch to `modeDelete` (only if list non-empty).
    - `esc`, `backspace` → emit `BackMsg{}` to return to main menu.
  - `tea.KeyPressMsg` (create mode):
    - printable runes → append to `input`.
    - `backspace` → trim last rune from `input`.
    - `enter` → validate name, emit `createProjectCmd(name)`.
    - `esc` → back to list mode.
  - `tea.KeyPressMsg` (delete mode):
    - `y` → emit `deleteProjectCmd(selectedName)`.
    - `n`, `esc` → back to list mode.
  - `createProjectCmd` / `deleteProjectCmd` — execute API call off main
    goroutine; on success re-emit `fetchProjectsCmd`; on error return `ErrMsg`.
- `UpdateTyped` helper (same pattern as `auth.Model`).
- `View() tea.View`:
  - Title bar.
  - Paginated list (current page slice); selected item highlighted.
  - Status bar with page indicator and shortcuts.
  - If `modeCreate`: inline input row at bottom ("New project name: <input>_").
  - If `modeDelete`: inline confirm row ("Delete '<name>'? [y/n]").
  - If `err != nil`: styled error line.
  - If `loading`: "Loading…" indicator.

Create `internal/tui/projects/projects_test.go`:
- Table-driven tests.
- Cover: `FetchedMsg` updates list; cursor movement; pagination boundary;
  `modeCreate` input accumulation and submit; `modeDelete` confirm and cancel;
  `ErrMsg` sets error field; `BackMsg` on `esc`.
- Use a stub `client.Client` via interface or `httptest` server.

---

## Task 3 — Remove "Authenticate" menu item (`internal/tui/root/menu.go`)

- Remove the `{ID: "authenticate", Label: "[A] Authenticate"}` entry from
  `defaultMenuItems()`.
- Remove the `case "authenticate":` branch from `Update`.

---

## Task 4 — Integrate Projects screen into root model (`internal/tui/root/`)

In `root.go`:
- Add `screenProjects screen` constant.
- Add `projectsModel projects.Model` field to `Model`.
- In `Update`, handle `case "projects":` in the menu enter path:
  - If `!m.session.IsAuthenticated()` → send `triggerAuthMsg{returnTo: screenProjects}`.
  - Else → `m.screen = screenProjects`, `m.projectsModel = projects.New(newClient(m), m.theme)`, return `m.projectsModel.Init()`.
- Handle `auth.AuthSuccessMsg`: after setting session, if `m.pendingScreen == screenProjects` → also initialise `m.projectsModel` and return its `Init()`.
- Handle `projects.BackMsg{}` → `m.screen = screenMain`.
- Delegate `screenProjects` in the `Init`, `Update`, `View` switches.
- Add `newClient(m Model) *client.Client` helper constructing the client from
  `m.profile.ServerURL` and `&m.session`.

---

## Task 5 — Unit tests: client package

_(Covered in Task 1 — listed separately to track test-only work.)_

- `TestListProjects_Success`
- `TestListProjects_Unauthorized`
- `TestCreateProject_Success`
- `TestCreateProject_Conflict`
- `TestDeleteProject_Success`
- `TestDeleteProject_NotFound`
- `TestClient_NetworkError`

---

## Task 6 — Unit tests: projects model

_(Covered in Task 2 — listed separately.)_

- `TestProjectsModel_FetchUpdates`
- `TestProjectsModel_CursorWrap`
- `TestProjectsModel_Pagination`
- `TestProjectsModel_CreateFlow`
- `TestProjectsModel_CreateValidation` (empty name rejected)
- `TestProjectsModel_DeleteConfirm`
- `TestProjectsModel_DeleteCancel`
- `TestProjectsModel_ErrDisplayed`
- `TestProjectsModel_BackOnEsc`

