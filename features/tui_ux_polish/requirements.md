# requirements.md — tui_ux_polish

## Goal

Improve the overall TUI user experience by introducing reusable UI primitives
(spinner, error modal, footer) and fixing cosmetic issues (non-uniform background
in HackerTheme). All existing screens are updated to use the new components.

---

## Scope

### In scope
- Generic, reusable `spinner` component with animated frames driven by `tea.Tick`.
- Generic, reusable `footer` component that renders a fixed keyboard-hints bar.
- Generic, reusable `modal` component for blocking error dialogs.
- Uniform terminal background: root model tracks `tea.WindowSizeMsg` and wraps
  every screen view in a full-size background-coloured container.
- All existing screens (`projects`, `secrets`, `profiles`, `auth`, `wizard`,
  `mainModel`, `profileSelect`) updated to replace ad-hoc loading/error/hint
  rendering with the new components.
- Fix silent failure in `root.go` when `session.SetAuthenticated` returns an
  error (unsupported key type): show an error modal instead of silently
  returning to the main menu.
- Add `SpinnerStyle()`, `FooterStyle()`, and `ModalStyle()` methods to the
  `Theme` interface, implemented in both `AdaptiveTheme` and `HackerTheme`.

### Out of scope
- Animated transitions between screens.
- Any new functional features (no new API calls, no new screens).
- Clipboard or encryption changes.
- CLI client.

---

## Functional Requirements

### FR-1 Spinner
- Package `internal/tui/spinner`.
- Exported `Model` with `Init() tea.Cmd`, `Update(tea.Msg) (Model, tea.Cmd)`,
  `View() string`.
- Animation: cycles through a set of frames (e.g. `⠋ ⠙ ⠹ ⠸ ⠼ ⠴ ⠦ ⠧ ⠇ ⠏`) at
  ~100 ms per frame using `tea.Tick`.
- `New(t theme.Theme) Model` constructor.
- All screens that had `loading bool` + static `"Loading…"` text use the
  spinner instead.

### FR-2 Footer
- Package `internal/tui/footer`.
- Exported `Model` with `View(hints string) string` (stateless, no `Init`/`Update`
  needed since it only renders).
- `New(t theme.Theme) Model` constructor.
- Renders hints text in `theme.FooterStyle()`.
- All screens replace their ad-hoc `StatusBarStyle()` hints line with the
  footer component.

### FR-3 Error Modal
- Package `internal/tui/modal`.
- Exported `Model` with `Init() tea.Cmd`, `Update(tea.Msg) (Model, tea.Cmd)`,
  `View(width int) string`.
- Constructor `New(t theme.Theme, message string) Model`.
- Blocks user interaction until the user presses `Enter` or `Esc` to dismiss.
- Emits `modal.DismissMsg{}` on dismiss.
- Rendered as a centred box with a title bar ("Error"), the error message, and
  a "[ Press Enter to dismiss ]" hint.
- All screens replace their inline `m.err` text rendering with the modal; the
  screen's `mode`/`state` machine gains a `modeError` (or equivalent) state.

### FR-4 Uniform Background
- `root.Model` gains `width int` and `height int` fields, populated by handling
  `tea.WindowSizeMsg` in `Update`.
- `root.Model.View()` wraps the active sub-screen's rendered string in a
  `lipgloss.Place(width, height, …)` call using `theme.BackgroundColor()` as
  background, so the entire terminal cell grid is filled.
- This fixes the HackerTheme visual artefact where non-rendered cells show the
  terminal's default background instead of `#0D0208`.

### FR-5 Theme Interface Extension
New methods added to `theme.Theme` and implemented by both `AdaptiveTheme` and
`HackerTheme`:

| Method | Purpose |
|---|---|
| `SpinnerStyle() lipgloss.Style` | style for the spinner frame glyph |
| `FooterStyle() lipgloss.Style` | style for the footer bar text |
| `ModalStyle() lipgloss.Style` | style for the modal box border and text |
| `DetailStyle() lipgloss.Style` | style for profile detail panel label/value text |
| `TableHeaderStyle() lipgloss.Style` | style for the secrets table header row and separator |

### FR-6 Profiles — Split Layout (list + detail panel)
- In `modeList` the Profiles screen is split into two panes side by side:
  - **Left pane** (~30 % of available width, minimum 20 cols): scrollable list
    of profile names. The active profile is marked with `*`. The currently
    highlighted entry is rendered with `SelectedMenuItemStyle()`.
  - **Right pane** (remaining width): detail panel for the highlighted profile,
    showing each field on its own line as `Label : value`, e.g.:
    ```
    Name          : production *
    Server URL    : https://ekvs.example.com
    Identity file : ~/.ssh/id_ed25519
    Theme         : hacker
    ```
    The panel updates immediately as the cursor moves through the list.
  - The two panes are joined horizontally with `lipgloss.JoinHorizontal`.
- In all other modes (`modeCreate`, `modeEdit`, `modeDeleteConfirm`,
  `modeReloadPrompt`) the existing full-width layout is kept unchanged.
- The `Model` receives `tea.WindowSizeMsg` so it knows the available width.
  Root must forward `WindowSizeMsg` to the active sub-model. Alternatively,
  root passes `width` when constructing the profiles model — the chosen
  approach must be documented as a decision.
- New theme method: `DetailStyle() lipgloss.Style` — used for the label/value
  text in the detail panel. Added to the `Theme` interface and implemented for
  both `AdaptiveTheme` and `HackerTheme`.

### FR-8 Wizard — SSH Key Discovery at `stepIdentityFile`
- The SSH key discovery logic currently private to `profiles.Model` is extracted
  into `internal/tui/config` as an exported function `DiscoverSSHKeys`, so that
  both `profiles` and `wizard` share a single implementation.
- `config.DiscoverSSHKeys(sshDirFn func() (string, error)) []string`:
  - If `sshDirFn` is nil, falls back to `config.SSHDir`.
  - Skips `.pub` files and the fixed ignore-list (`known_hosts`, `known_hosts.old`,
    `config`, `authorized_keys`).
  - Returns nil when the directory cannot be read.
- `profiles.Model.discoverSSHKeys()` is removed; call sites replaced with
  `tuiconfig.DiscoverSSHKeys(m.sshDirFn)`.
- The wizard gains the same dual-mode identity step as the Profiles create form:
  - On entering `stepIdentityFile`, call `tuiconfig.DiscoverSSHKeys(m.sshDirFn)`.
  - If keys are found, show a scrollable pick list with `SelectedMenuItemStyle()`.
    An option `"m — enter a custom path"` is always shown at the bottom.
  - If no keys are found, fall back directly to manual text input.
  - In pick mode: `↑`/`k` and `↓`/`j` navigate; `Enter` confirms; `m` switches
    to manual mode.
  - In manual mode: `Esc` (when keys were discovered) returns to pick mode;
    otherwise goes to the previous step.
  - `sshDirFn func() (string, error)` field on `wizard.Model` can be overridden
    for tests (no real `~/.ssh` access in unit tests).
- Goal: the wizard UX is consistent with `modeCreate` in the Profiles screen,
  with zero code duplication.

### FR-7 Secrets — Tabular Display
- The secrets list in `modeList` is rendered as a proper table:
  - Header row: `KEY` | `VALUE` with a horizontal separator line beneath.
  - Each row: left-aligned KEY column (width = max key length), right-aligned
    or left-aligned VALUE column.
  - The selected row is highlighted with `SelectedMenuItemStyle()`.
  - In `modeAdd` and `modeEdit` the existing inline input form is kept
    unchanged.
- The table is built with plain `lipgloss` (no external table library); a thin
  vertical separator (`│`) is placed between KEY and VALUE columns.
- New theme method: `TableHeaderStyle() lipgloss.Style` — used for the header
  row and separator line. Added to the `Theme` interface and implemented for
  both `AdaptiveTheme` and `HackerTheme`.

---

## Non-Functional Requirements

- NFR-1: `go build ./...` must succeed after every individual task.
- NFR-2: `go test ./...` must pass (no regressions) after every individual task.
- NFR-3: New packages (`spinner`, `footer`, `modal`) must have `*_test.go` files
  with table-driven unit tests.
- NFR-4: No external dependencies beyond those already in `go.mod`
  (`charm.land/bubbletea/v2`, `github.com/charmbracelet/lipgloss`).

---

## Decisions

| # | Decision |
|---|---|
| D-1 | Spinner is a stateful bubbletea sub-model using `tea.Tick` (no third-party library). |
| D-2 | Footer is stateless — only `View(hints string) string` is needed; no `Init`/`Update`. |
| D-3 | Error modal is a separate package (`internal/tui/modal`) so it can be reused by any screen. |
| D-4 | Background fill is applied at the root level using `lipgloss.Place`; sub-screens are unaware. |
| D-5 | `AdaptiveTheme` background fill uses `adaptiveBackground` but `lipgloss.Place` is a no-op when `BackgroundColor` is transparent — acceptable for light/dark adaptive terminals. |
| D-6 | Root forwards `tea.WindowSizeMsg` to the active sub-model (same dispatch loop as key presses). Profiles `Model` gains a `width int` field; root re-creates or updates the model on resize. |
| D-7 | Secrets tabular display is implemented with plain `lipgloss` string formatting (no external table library). The `│` separator is a single Unicode character; no box-drawing borders are used. |
| D-8 | SSH key discovery is extracted into `config.DiscoverSSHKeys(sshDirFn)` (package `internal/tui/config`) and shared by both `profiles` and `wizard`. The `sshDirFn` parameter (nil → `SSHDir`) enables test injection without touching real `~/.ssh`. |

---

## Files To Create

```
internal/tui/spinner/spinner.go
internal/tui/spinner/spinner_test.go
internal/tui/footer/footer.go
internal/tui/footer/footer_test.go
internal/tui/modal/modal.go
internal/tui/modal/modal_test.go
internal/tui/wizard/wizard_test.go
```

## Files To Modify

```
internal/tui/config/config.go        — add DiscoverSSHKeys
internal/tui/config/config_test.go   — tests for DiscoverSSHKeys
internal/tui/theme/theme.go          — add SpinnerStyle, FooterStyle, ModalStyle
internal/tui/theme/theme_test.go     — test new methods
internal/tui/root/root.go            — track WindowSizeMsg, wrap View, fix auth error
internal/tui/projects/projects.go    — use spinner, footer, modal
internal/tui/secrets/secrets.go      — use spinner, footer, modal
internal/tui/profiles/profiles.go    — remove discoverSSHKeys, use config.DiscoverSSHKeys; use footer, modal
internal/tui/profiles/profiles_test.go — verify no regression
internal/tui/auth/auth.go            — use modal instead of stateError
internal/tui/wizard/wizard.go        — add dual-mode identity step using config.DiscoverSSHKeys
internal/tui/root/menu.go            — use footer
internal/tui/root/profileselect.go   — use footer
```




