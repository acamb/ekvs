# plan.md — tui_ux_polish

Ordered list of atomic implementation tasks. Each task must leave the codebase
in a state where `go build ./...` and `go test ./...` pass before moving to
the next one.

---

## Task 1 — Extend Theme interface

**Description**  
Add five new methods to the `Theme` interface in `internal/tui/theme/theme.go`
and provide implementations for both `AdaptiveTheme` and `HackerTheme`:
- `SpinnerStyle() lipgloss.Style`
- `FooterStyle() lipgloss.Style`
- `ModalStyle() lipgloss.Style`
- `DetailStyle() lipgloss.Style` — used for the profile detail panel
- `TableHeaderStyle() lipgloss.Style` — used for the secrets table header/separator

Update `theme_test.go` to cover all five new methods.

**Files**
- `internal/tui/theme/theme.go`
- `internal/tui/theme/theme_test.go`

---

## Task 2 — Track terminal size in root model and propagate to sub-models

**Description**  
Add `width int` and `height int` fields to `root.Model`. Handle
`tea.WindowSizeMsg` at the top of `root.Model.Update` (before the screen
dispatch switch) to keep these fields up to date. After updating the fields,
also forward the `tea.WindowSizeMsg` to the currently active sub-model via
the normal dispatch so that sub-models that need the width (e.g. profiles)
receive it. No visual change yet.

**Files**
- `internal/tui/root/root.go`

---

## Task 3 — Create `internal/tui/spinner` package

**Description**  
Implement a generic bubbletea sub-model that cycles through Unicode braille
spinner frames (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) at 100 ms per frame using `tea.Tick`.
Exports: `Model`, `New(t theme.Theme) Model`, `TickMsg`, `Init()`,
`Update()`, `UpdateTyped()`, `View() string`.
Write table-driven unit tests in `spinner_test.go`.

**Files**
- `internal/tui/spinner/spinner.go`
- `internal/tui/spinner/spinner_test.go`

---

## Task 4 — Create `internal/tui/footer` package

**Description**  
Implement a stateless footer helper. Exports: `Model`, `New(t theme.Theme) Model`,
`View(hints string) string`.
The hints string is rendered using `theme.FooterStyle()`.
Write unit tests.

**Files**
- `internal/tui/footer/footer.go`
- `internal/tui/footer/footer_test.go`

---

## Task 5 — Create `internal/tui/modal` package

**Description**  
Implement a blocking error modal sub-model. Constructor:
`New(t theme.Theme, message string) Model`.
Pressing `Enter` or `Esc` emits `modal.DismissMsg{}` and returns `tea.Quit`
for the modal's own command (the parent model handles the state transition).
`View(width int) string` renders a centred box using `theme.ModalStyle()` with:
- Title bar: `" Error "`
- Body: the error message (word-wrapped to `width-4`)
- Footer hint: `"[ Press Enter to dismiss ]"`

Exports: `Model`, `DismissMsg`, `New`, `Init`, `Update`, `UpdateTyped`, `View`.
Write unit tests.

**Files**
- `internal/tui/modal/modal.go`
- `internal/tui/modal/modal_test.go`

---

## Task 6 — Update `projects` screen (spinner + footer + modal)

**Description**  
Replace the ad-hoc loading/error/hint rendering in `projects/projects.go`:
- Replace `loading bool` + static `"Loading…"` with a `spinner.Model` field;
  forward `spinner.TickMsg` in `Update`; propagate `spinner.Init()` from
  `projects.Init()`.
- Add a `footer.Model` field; replace the manual `StatusBarStyle().Render(…)`
  line with `m.footer.View(hints)`.
- Add a `modeError mode` constant and a `modalModel modal.Model` field; when
  `ErrMsg` arrives, switch to `modeError` and populate `modalModel`; in
  `Update` forward messages to the modal when in `modeError`; on
  `modal.DismissMsg` return to `modeList`.
- In `View()` when `modeError`, render the modal overlay.

**Files**
- `internal/tui/projects/projects.go`
- `internal/tui/projects/projects_test.go` (update/add tests)

---

## Task 7 — Update `secrets` screen (spinner + footer + modal + tabular display)

**Description**  
Same structural changes as Task 6, plus tabular list rendering:
- `spinner.Model` replaces `loading bool`.
- `footer.Model` replaces manual hints line.
- `modeError` + `modal.Model` replace inline error text.
- In `modeList`, rewrite `View()` to render a proper table:
  - Header row: `KEY` | `VALUE (decrypted)` styled with `theme.TableHeaderStyle()`.
  - Horizontal separator line (e.g. `─────────┼──────────`) also styled with
    `theme.TableHeaderStyle()`.
  - Each data row: left-aligned KEY column (padded to max key length), `│`
    separator, VALUE column. Selected row uses `SelectedMenuItemStyle()`.

**Files**
- `internal/tui/secrets/secrets.go`
- `internal/tui/secrets/secrets_test.go` (update/add tests)

---

## Task 8 — Secrets screen: key search/filter

**Description**  
Add an incremental search mode (`modeSearch`) to the secrets screen that allows
the user to filter the displayed rows by key substring.

- Add `modeSearch mode` constant.
- Add `searchQuery string` field to `secrets.Model`.
- In `modeList`, pressing `/` switches to `modeSearch` and clears `searchQuery`.
- In `modeSearch`:
  - Typing any printable character appends to `searchQuery`; `Backspace` removes
    the last rune.
  - The bubbles table is rebuilt on every keystroke with only the rows whose
    `Key` contains `searchQuery` (case-insensitive, using
    `strings.Contains(strings.ToLower(key), strings.ToLower(query))`).
  - `cursor` is reset to 0 whenever the filtered set changes.
  - `Enter` or `Esc` exits `modeSearch`, returns to `modeList` keeping the
    current filter active so the table stays filtered; a second `Esc` (while
    already in `modeList` with a non-empty query) clears the filter and rebuilds
    the full table.
  - The footer hint line shows: `"Search: <query>█  Enter/Esc confirm • Esc clear"`.
- `pageSecrets()` respects the active `searchQuery`: when non-empty it filters
  `m.secrets` before slicing the page window; pagination (`totalPages`) is
  recomputed accordingly.
- `buildTable()` requires no changes beyond receiving the already-filtered slice
  from `pageSecrets()`.
- Footer hint in `modeList` gains: `• / search` (and `• Esc clear filter` when
  `searchQuery != ""`).

**Files**
- `internal/tui/secrets/secrets.go`
- `internal/tui/secrets/secrets_test.go` (add tests for modeSearch transitions,
  filtering behaviour, cursor reset, and clear-on-double-Esc)

---

## Task 9 — Update `profiles` screen (footer + modal + split layout)

**Description**  
- Add `footer.Model` field; replace manual hints with `m.footer.View(hints)`.
- Add `modeError` state + `modal.Model`; show modal on errors (config save
  failure, etc.); dismiss returns to the previous mode.
- Add `width int` field to `profiles.Model`; handle `tea.WindowSizeMsg` in
  `Update` (forwarded from root, see Task 2).
  The root model must inject the already-known terminal width immediately after
  constructing `profiles.Model` (in `triggerProfilesMsg` and `ConfigChangedMsg`
  handlers) by calling `m.profilesModel.UpdateTyped(tea.WindowSizeMsg{Width: m.width, Height: m.height})`
  before assigning to `m.profilesModel`. This ensures the split layout is
  visible on first render without requiring a resize event.
- In `modeList`, rewrite `View()` to render a two-pane split layout with a
  vertical separator:
  - Left pane width = `max(20, width * 30 / 100)`.
  - Separator = 3 chars wide: ` │ ` (space + U+2502 BOX DRAWINGS LIGHT VERTICAL + space),
    rendered as a column of `│` characters repeated for the height of the tallest pane.
  - Right pane width = `width - leftW - 3`.
  - Left pane: list of profile names; active profile marked with `*`; cursor
    highlighted with `SelectedMenuItemStyle()`; each entry on its own line.
  - Right pane: detail panel for the highlighted profile rendered with
    `theme.DetailStyle()`:
    ```
    Name          : production *
    Server URL    : https://ekvs.example.com
    Identity file : ~/.ssh/id_ed25519
    Theme         : hacker
    ```
  - Panes joined with `lipgloss.JoinHorizontal(lipgloss.Top, leftPane, sep, rightPane)`.
  - When `width == 0` (before first `WindowSizeMsg`), fall back to the old
    single-column list view to avoid layout artefacts.
- All other modes (`modeCreate`, `modeEdit`, `modeDeleteConfirm`,
  `modeReloadPrompt`) keep the existing full-width layout unchanged.

**Files**
- `internal/tui/profiles/profiles.go`
- `internal/tui/profiles/profiles_test.go` (update/add tests)

---

## Task 10 — Update `auth` screen (modal)

**Description**  
Replace `stateError`/`errMsg string` in `auth/auth.go` with `modal.Model`:
- Remove `stateError` constant and `errMsg` field.
- Add `modalModel modal.Model` and `showModal bool` fields.
- When an error occurs (key load failure, wrong passphrase, etc.) populate
  `modalModel` and set `showModal = true`.
- Forward messages to `modalModel` when `showModal`; on `modal.DismissMsg`
  set `showModal = false` and allow retry (stay on passphrase prompt).

**Files**
- `internal/tui/auth/auth.go`
- `internal/tui/auth/auth_test.go` (update/add tests)
- `internal/tui/auth/sign.go` (if error display touched)

---

## Task 11 — Fix auth error in root model

**Description**  
In `root.go`, when `session.SetAuthenticated` returns an error (unsupported key
type), currently the code silently redirects to `screenMain`. Instead:
- Add `showModal bool` and `modalModel modal.Model` fields to `root.Model`.
- Show the error modal overlaid on top of `screenMain`.
- Forward messages to `modalModel` when `m.showModal`; on `modal.DismissMsg`
  clear `showModal`.
- In `View()` when `m.showModal`, render the modal centred using `m.width`.

**Files**
- `internal/tui/root/root.go`
- `internal/tui/root/root_test.go` (update/add tests)

---

## Task 12 — Update `wizard`, `mainModel`, `profileSelect` (footer)

**Description**  
Add `footer.Model` to `wizard.Model`, `mainModel`, and `profileSelectModel`.
Replace their manual `StatusBarStyle().Render(…)` or equivalent hints lines
with `m.footer.View(hints)`.

**Files**
- `internal/tui/wizard/wizard.go`
- `internal/tui/root/menu.go`
- `internal/tui/root/profileselect.go`

---

## Task 13 — Uniform background fill in root View

**Description**  
In `root.Model.View()`, after delegating to the active sub-screen, wrap the
returned string with:
```go
lipgloss.NewStyle().
    Background(m.theme.BackgroundColor()).
    Width(m.width).
    Height(m.height).
    Render(content)
```
This ensures `HackerTheme`'s `#0D0208` background fills the entire terminal,
eliminating the visual artefact of the terminal's default background showing
through. Guard against `width == 0` (before first `WindowSizeMsg`) by returning
the raw content unchanged.

**Files**
- `internal/tui/root/root.go`

---

## Task 14 — Final review, tests and build verification

**Description**  
- Run `go build ./...` and `go test ./...` and fix any remaining issues.
- Ensure all new packages have adequate test coverage (spinner frames cycling,
  footer rendering, modal dismiss flow).
- Remove any dead code left over from the refactor (e.g. unused `stateError`,
  old `err error` fields that are now handled by modal, direct `ErrorStyle()`
  inline calls).

**Files**
- Any file with residual issues found during review.





