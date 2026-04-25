# plan.md — tui_secrets

Ordered task list. Each task must pass `make test` (no regressions) before the next begins.

---

## Task 1 — Add clipboard dependency

- Run `go get golang.design/x/clipboard` to add the dependency to `go.mod` / `go.sum`.
- Verify the module is available: `go build golang.design/x/clipboard`.

---

## Task 2 — Extend `internal/tui/client` with secrets API

Files: `internal/tui/client/client.go`, `internal/tui/client/errors.go`

- Define `SecretEntry struct { Key, Value string }` in `client.go`.
- Implement `ListSecrets(project string) ([]SecretEntry, error)`:
  - `GET /projects/{project}/secrets` → 200 with `{"secrets": [{"key":"…","value":"…"}]}`.
  - Maps 401 → `ErrUnauthorized`, 404 → `ErrNotFound`, other non-200 → `ServerError`.
- Implement `SetSecret(project, key, value string) error`:
  - `PUT /projects/{project}/secrets/{key}` with JSON body `{"value": value}` → 200.
- Implement `DeleteSecret(project, key string) error`:
  - `DELETE /projects/{project}/secrets/{key}` → 204.

Unit tests in `client_test.go` (table-driven, httptest.NewServer fakes).

---

## Task 3 — Add `OpenSecretsMsg` to `internal/tui/projects`

Files: `internal/tui/projects/messages.go`, `internal/tui/projects/projects.go`

- Add `OpenSecretsMsg{ Project string }` to `messages.go`.
- In `projects.go`, handle `Enter` key in `modeList`: emit `OpenSecretsMsg{Project: selected}`.
- Add/update unit test `TestProjectsModel_EnterOpensSecrets`.

---

## Task 4 — Implement `internal/tui/secrets` package

New files: `internal/tui/secrets/messages.go`, `internal/tui/secrets/secrets.go`

### 4a — `messages.go`
Define `BackMsg`, `FetchedMsg{Secrets []client.SecretEntry}`, `ErrMsg{Err error}`.

### 4b — `secrets.go`

Implement `Model` with:
- Fields: `project string`, `client apiClient`, `sess *session.Session`, `theme theme.Theme`, `secrets []client.SecretEntry`, `cursor int`, `page int`, `mode mode`, `inputKey string`, `inputValue string`, `activeField int` (0=key, 1=value), `err error`, `loading bool`.
- `New(project, client, sess, theme)` constructor.
- `Init()` → returns `fetchCmd()`.
- `Update(msg)`:
  - `FetchedMsg` → populate `secrets`, clear `loading`.
  - `ErrMsg` → set `err`, clear `loading`.
  - Key events dispatched by `mode`.
- `View()`:
  - Header: `"Project: {project}"`.
  - Table with columns `KEY` and `VALUE (decrypted)` in `modeList`.
  - Inline input prompt in `modeAdd`/`modeEdit`.
  - Inline confirmation prompt in `modeDelete`.
  - Error line at the bottom when `err != nil`.
  - Footer: keyboard hints (context-sensitive per mode).

---

## Task 5 — Update `internal/tui/root` for navigation

File: `internal/tui/root/root.go` (and any related navigation files)

- Add a `secrets.Model` slot to the root navigation model.
- On `projects.OpenSecretsMsg`: construct `secrets.New(…)`, call `Init()`, switch active screen to secrets.
- On `secrets.BackMsg`: return to projects screen, re-trigger project list fetch.

---

## Task 6 — Unit tests for `internal/tui/secrets`

File: `internal/tui/secrets/secrets_test.go`

Write table-driven tests (see `validation.md` for the full required test matrix).
Use a fake `apiClient` interface and a real `session.Session` (call `SetAuthenticated` with a test key).

---

## Task 7 — Final check

- `make test` passes with no regressions.
- `go build ./cmd/tui/...` succeeds.
- Manual smoke-test per `validation.md` sections 3–5.

