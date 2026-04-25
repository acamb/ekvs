# requirements.md — tui_secrets

## Goal

Implement the secret management screen in the TUI: from the Projects screen the
user selects a project and navigates to a Secrets screen where they can list,
view, add, edit, and delete key-value pairs. Values are transparently
encrypted/decrypted via `session.Session`.

---

## Scope

### In scope

#### `internal/tui/client` — new API methods
- `ListSecrets(project string) ([]SecretEntry, error)` — `GET /projects/{name}/secrets`
- `SetSecret(project, key, value string) error` — `PUT /projects/{name}/secrets/{key}` with JSON body `{"value": "<encrypted-blob>"}`
- `DeleteSecret(project, key string) error` — `DELETE /projects/{name}/secrets/{key}`

`SecretEntry` is a value type `{ Key, Value string }` defined in the `client` package.
The `value` field transported to/from the server is always the **encrypted blob** (base64-encoded AES-GCM ciphertext). Encryption/decryption is the responsibility of the caller (the `secrets` model), not the client.

#### `internal/tui/secrets` — new package (bubbletea model)
- `Model` struct with modes: `modeList`, `modeAdd`, `modeEdit`, `modeDelete`.
- Receives the project name and a `*session.Session` pointer at construction.
- On `Init`: fetches secret list via `ListSecretsCmd`.
- **modeList** behaviour:
  - Renders a two-column table: `KEY` | `VALUE (decrypted)`. Values are decrypted inline using `session.Decrypt`; if decryption fails the cell shows `<error>`.
  - Cursor movement: `↑`/`↓` (wraps), `PgUp`/`PgDn` or `←`/`→` for pagination (pageSize = 10, consistent with `tui_projects`).
  - `n` → switch to `modeAdd` (empty key/value inputs).
  - `e` → switch to `modeEdit` (pre-fill key (read-only) and decrypted value into inputs).
  - `d` → switch to `modeDelete` (inline y/n confirmation).
  - `c` → copy decrypted value of selected secret to system clipboard via `golang.design/x/clipboard`.
  - `esc` / `q` → emit `BackMsg` (return to Projects screen).
- **modeAdd** behaviour:
  - Two sequential text fields: first `KEY`, then `VALUE` (both visible, not masked).
  - `Tab` / `Enter` advances from KEY field to VALUE field.
  - `Enter` on VALUE field: encrypt value with `session.Encrypt`, call `SetSecret`, re-fetch list, return to `modeList`.
  - `Esc` cancels, returns to `modeList`.
- **modeEdit** behaviour:
  - KEY field is displayed but not editable (read-only label).
  - VALUE field is pre-filled with the **decrypted** current value and is editable.
  - `Enter` on VALUE field: encrypt new value, call `SetSecret`, re-fetch, return to `modeList`.
  - `Esc` cancels, returns to `modeList`.
- **modeDelete** behaviour:
  - Inline prompt `Delete "<key>"? [y/N]`.
  - `y` / `Y`: call `DeleteSecret`, re-fetch, return to `modeList`.
  - Any other key (including `Esc`, `n`, `Enter`): cancel, return to `modeList`.
- Error handling: on any API or crypto error, set `err` field; `View()` renders it as a single error line at the bottom; next keypress clears it.
- Loading state: `loading bool` field; while `true`, show a spinner/placeholder row.

#### `internal/tui/projects` — navigation update
- When the user presses `Enter` on a selected project, emit `OpenSecretsMsg{Project: name}` instead of doing nothing.
- The root model handles `OpenSecretsMsg` by pushing the Secrets screen onto the navigation stack.

#### `internal/tui/root` — navigation update
- Add a `secrets.Model` to the root navigation stack.
- Handle `OpenSecretsMsg` (from projects model): instantiate `secrets.New(project, client, sess, theme)` and switch the active model.
- Handle `secrets.BackMsg`: return to the Projects screen (re-trigger project list fetch).

#### Clipboard dependency
- Add `golang.design/x/clipboard` to `go.mod` / `go.sum`.
- Clipboard write is best-effort: if the library returns an error (e.g. no display), show it as the inline error string; do not crash.

### Out of scope
- Pagination of secrets beyond `pageSize = 10` per page (can be added in `tui_ux_polish`).
- Secret key rename (would require delete + re-add).
- Batch delete.
- Value masking/hiding toggle (deferred to `tui_ux_polish`).
- CLI client (separate milestones).
- SSH agent support (Phase 6).

---

## Decisions

| # | Decision |
|---|----------|
| 1 | Decrypted values are shown **inline** as a second column in the list (choice 1a). No separate detail view. |
| 2 | Clipboard copy uses `golang.design/x/clipboard`. Errors are shown inline; they do not interrupt the flow. |
| 3 | In `modeEdit` the current encrypted blob is decrypted and pre-filled into the VALUE input so the user can edit the existing value. |
| 4 | Navigation from Projects → Secrets follows the established pattern: `OpenSecretsMsg` emitted by the projects model, handled by the root model. |
| 5 | Value input is **visible** (not masked) in both `modeAdd` and `modeEdit`. |
| 6 | The `client` package transports encrypted blobs only; encryption/decryption lives exclusively in the `secrets` model, which owns a `*session.Session`. |
| 7 | `SecretEntry` is defined in `internal/tui/client` alongside the existing API types. |

---

## Updated `client.Client` methods (signatures)

```go
type SecretEntry struct {
    Key   string
    Value string // encrypted blob as returned by the server
}

func (c *Client) ListSecrets(project string) ([]SecretEntry, error)
func (c *Client) SetSecret(project, key, value string) error   // value = encrypted blob
func (c *Client) DeleteSecret(project, key string) error
```

## `secrets.Model` constructor

```go
// New returns a Model for the given project, ready to be Init()-ed.
func New(project string, c *client.Client, sess *session.Session, t theme.Theme) Model
```

## Messages

```go
// messages.go in internal/tui/secrets
type BackMsg struct{}

type FetchedMsg struct {
    Secrets []client.SecretEntry
}

type ErrMsg struct {
    Err error
}
```

```go
// added to internal/tui/projects/messages.go
type OpenSecretsMsg struct {
    Project string
}
```

