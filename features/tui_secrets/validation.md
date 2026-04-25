# validation.md — tui_secrets

## Acceptance criteria

The feature is considered complete when **all** of the following criteria are met.

---

## 1. Unit tests

```bash
make test
```

- `ok  ekvs/internal/tui/client` — all new secrets-related tests pass.
- `ok  ekvs/internal/tui/secrets` — statement coverage ≥ 85%.
- `ok  ekvs/internal/tui/projects` — `TestProjectsModel_EnterOpensSecrets` passes.
- No regressions on any other package.

Or targeted:
```bash
go test ./internal/tui/... -v -cover
```

### Required tests — `internal/tui/client`

| Test name | What it verifies |
|-----------|-----------------|
| `TestListSecrets_Success` | `GET /projects/p/secrets` returns `{"secrets":[{"key":"k","value":"v"}]}` → `[]SecretEntry{{Key:"k",Value:"v"}}, nil` |
| `TestListSecrets_Empty` | Server returns `{"secrets":[]}` → empty slice, no error |
| `TestListSecrets_Unauthorized` | Server returns 401 → `ErrUnauthorized` |
| `TestListSecrets_NotFound` | Server returns 404 → `ErrNotFound` |
| `TestSetSecret_Success` | `PUT /projects/p/secrets/k` returns 200 → no error |
| `TestSetSecret_NotFound` | Server returns 404 → `ErrNotFound` |
| `TestSetSecret_BadRequest` | Server returns 400 → `ServerError` |
| `TestDeleteSecret_Success` | `DELETE /projects/p/secrets/k` returns 204 → no error |
| `TestDeleteSecret_NotFound` | Server returns 404 → `ErrNotFound` |

### Required tests — `internal/tui/secrets`

| Test name | What it verifies |
|-----------|-----------------|
| `TestSecretsModel_FetchUpdates` | `FetchedMsg{[{Key:"k",Value:"<blob>"}]}` populates `secrets`, clears `loading` |
| `TestSecretsModel_ErrDisplayed` | `ErrMsg{err}` sets `err`; `View()` contains error text |
| `TestSecretsModel_ErrClearedOnKeypress` | After `ErrMsg`, any key clears `err` |
| `TestSecretsModel_CursorDown` | `↓` moves cursor 0→1; wraps at page end |
| `TestSecretsModel_CursorUp` | `↑` wraps from 0 → last item on page |
| `TestSecretsModel_Pagination` | 15 secrets, pageSize=10: page 0 shows items 0–9; `→` shows 10–14 |
| `TestSecretsModel_BackOnEsc` | `esc` emits `BackMsg` |
| `TestSecretsModel_AddMode` | `n` switches to `modeAdd`; typing accumulates in `inputKey`; `Tab` advances to `inputValue` |
| `TestSecretsModel_AddModeEscCancels` | `esc` in `modeAdd` returns to `modeList`, clears inputs |
| `TestSecretsModel_AddModeSubmit` | `Enter` on VALUE field: fake client receives `SetSecret` call with encrypted blob; `FetchedMsg` re-fetches list |
| `TestSecretsModel_EditMode` | `e` switches to `modeEdit`; `inputValue` is pre-filled with decrypted value of selected secret |
| `TestSecretsModel_EditModeSubmit` | `Enter` on VALUE field in `modeEdit`: `SetSecret` called with new encrypted blob |
| `TestSecretsModel_DeleteMode` | `d` switches to `modeDelete`; `View()` shows confirmation prompt |
| `TestSecretsModel_DeleteConfirm` | `y` in `modeDelete`: `DeleteSecret` called; list re-fetched |
| `TestSecretsModel_DeleteCancel` | `n` in `modeDelete`: returns to `modeList` without calling `DeleteSecret` |
| `TestSecretsModel_ViewDecryptsValues` | `View()` output contains decrypted plaintext, not the raw blob |
| `TestSecretsModel_ViewShowsErrorOnDecryptFail` | If session not authenticated, `View()` shows `<error>` in value column |

### Required tests — `internal/tui/projects`

| Test name | What it verifies |
|-----------|-----------------|
| `TestProjectsModel_EnterOpensSecrets` | With a project in the list and cursor at index 0, pressing `Enter` produces `OpenSecretsMsg{Project: "name"}` |

---

## 2. Build

```bash
go build ./cmd/tui/...
```

Must complete with no errors or warnings.

---

## 3. Smoke test — list and view secrets

Start the server and create a test project with at least one secret via the CLI or curl.

```bash
# (server running on :9090 with a known key)
curl -s http://127.0.0.1:9090/projects/demo/secrets   # should return the encrypted blob

./tui  # launch the TUI
```

- Log in with the SSH key.
- Navigate to the Projects screen; `demo` project is listed.
- Press `Enter` on `demo` → Secrets screen opens with the project header `Project: demo`.
- The secret key and **decrypted** value are shown in the table.

---

## 4. Smoke test — add a secret

From the Secrets screen:

- Press `n` → add form appears with `KEY:` and `VALUE:` prompts.
- Type a key name, press `Tab`, type a value, press `Enter`.
- The list refreshes and the new entry appears with the correct decrypted value.
- The raw value stored on disk (in `data/`) is an encrypted blob (not plaintext).

---

## 5. Smoke test — edit, delete, clipboard

- Press `e` on an existing secret → VALUE field is pre-filled with the decrypted value.
- Change the value, press `Enter` → list refreshes with the updated decrypted value.
- Press `d` → confirmation prompt appears; press `y` → secret is removed from the list.
- Press `c` on any secret → decrypted value is in the system clipboard (paste to verify).

