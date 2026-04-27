# validation.md — tui_ux_polish

Describes how to verify that the `tui_ux_polish` feature is correctly
implemented — both through automated tests and manual inspection.

---

## Automated checks (must all pass)

```
go build ./...
go test ./...
```

These must pass after **every individual task** in `plan.md`, not only at the end.

---

## Component checklists

### Spinner (`internal/tui/spinner`)

- [ ] `New(theme)` returns a `Model` without panicking.
- [ ] `Init()` returns a non-nil `tea.Cmd`.
- [ ] Sending a `spinner.TickMsg` via `Update` advances the frame index.
- [ ] Frame index wraps around after the last frame.
- [ ] `View()` returns a non-empty string styled with `theme.SpinnerStyle()`.
- [ ] Unit test: assert frame sequence over N ticks matches expected frames.

### Footer (`internal/tui/footer`)

- [ ] `New(theme)` returns a `Model` without panicking.
- [ ] `View("some hints")` returns a string containing `"some hints"`.
- [ ] `View("")` returns an empty or blank string (no crash).
- [ ] The rendered string uses `theme.FooterStyle()` (check background/foreground
  by inspecting the lipgloss-rendered ANSI escape sequences, or by testing
  that the raw hint text is present in the output).
- [ ] Unit test: table-driven, various hint strings including empty.

### Error Modal (`internal/tui/modal`)

- [ ] `New(theme, "some error")` returns a `Model` without panicking.
- [ ] `View(80)` contains the string `"some error"`.
- [ ] `View(80)` contains `"Error"` (title).
- [ ] `View(80)` contains `"Enter"` (dismiss hint).
- [ ] Sending `tea.KeyPressMsg{Code: tea.KeyEnter}` via `Update` emits
  `modal.DismissMsg{}` as a message (the command must fire `modal.DismissMsg{}`).
- [ ] Sending `tea.KeyPressMsg{Code: tea.KeyEsc}` via `Update` also emits
  `modal.DismissMsg{}`.
- [ ] No other key dismisses the modal.
- [ ] Unit test: table-driven, covering Enter, Esc, and a regular key press.

### Theme extension

- [ ] `AdaptiveTheme.SpinnerStyle()`, `FooterStyle()`, `ModalStyle()` compile
  and return non-zero `lipgloss.Style` values.
- [ ] `HackerTheme.SpinnerStyle()`, `FooterStyle()`, `ModalStyle()` compile
  and return non-zero `lipgloss.Style` values with `Background(hackerBackground)`.
- [ ] `theme_test.go` tests the new methods for both themes.

---

## Per-screen checklist

For each screen, verify after the refactor:

| Screen | Spinner replaces "Loading…" | Footer replaces manual hints | Modal replaces inline error | Split layout | Tabular display |
|---|---|---|---|---|---|
| `projects` | ✓ | ✓ | ✓ | n/a | n/a |
| `secrets` | ✓ | ✓ | ✓ | n/a | ✓ |
| `profiles` | n/a | ✓ | ✓ | ✓ | n/a |
| `auth` | n/a | n/a | ✓ | n/a | n/a |
| `wizard` | n/a | ✓ | n/a | n/a | n/a |
| `mainModel` (menu) | n/a | ✓ | n/a | n/a | n/a |
| `profileSelect` | n/a | ✓ | n/a | n/a | n/a |

### Profiles split layout (FR-6)

- [ ] `profiles.Model` gains a `width int` field.
- [ ] Sending a `tea.WindowSizeMsg{Width: 120, Height: 40}` via `Update` sets
  `m.width = 120`.
- [ ] In `modeList` with `width = 120`, `View()` renders two panes side by side
  (left ≈ 36 cols, right ≈ 84 cols); both panes are present in the rendered string.
- [ ] The right pane contains the fields of the currently highlighted profile
  (`Name`, `Server URL`, `Identity file`, `Theme`).
- [ ] Moving the cursor up/down updates the right pane to reflect the new
  highlighted profile.
- [ ] The active profile is marked with ` *` in the left pane.
- [ ] When `width == 0`, no panic occurs and the single-column fallback is used.
- [ ] Modes other than `modeList` are unaffected by the split layout.
- [ ] Unit test: render with `width = 80`, two profiles; assert right-pane text
  contains the highlighted profile's server URL.

### Secrets tabular display (FR-7)

- [ ] In `modeList` the `View()` output contains a header line with `KEY` and
  `VALUE`.
- [ ] The header is followed by a separator line containing `─` or `┼`
  characters.
- [ ] Data rows are aligned: KEY column width equals the longest key name.
- [ ] The selected row contains `>` or equivalent highlighting.
- [ ] Modes `modeAdd` and `modeEdit` are visually unchanged (inline form).
- [ ] Unit test: render with two secrets; assert header, separator, and both
  key names appear in the rendered output.

---

## Background fill

- [ ] `root.Model` gains `width` and `height` fields.
- [ ] A `tea.WindowSizeMsg` sent to `root.Update` updates both fields.
- [ ] With HackerTheme active, the rendered output of `root.View()` when
  `width > 0` is wrapped in a `lipgloss.Style` with `Background(#0D0208)`
  and the correct dimensions.
- [ ] When `width == 0` (before first `WindowSizeMsg`), `View()` returns the
  raw sub-screen content without panicking.

---

## Manual end-to-end test

Start the server and run the TUI (`make run-tui` or equivalent).

### Scenario A — Spinner
1. Open the TUI and navigate to **Projects**.
2. Observe that while the project list is loading, an animated spinner is shown
   (frames change visibly, not static `"Loading…"` text).

### Scenario B — Error modal
1. Start the TUI pointed at a server URL that is not reachable.
2. Navigate to **Projects**.
3. Observe that a centred modal dialog appears with title "Error", the network
   error message, and a dismiss hint.
4. Press `Enter` — modal closes, user returns to the project list (empty).
5. Press `Esc` — same behaviour.

### Scenario C — Footer
1. Navigate to any screen (Projects, Secrets, Profiles, Main Menu).
2. Observe that a consistent footer bar is visible at the bottom of the screen
   showing context-sensitive keyboard hints.
3. Hints change correctly when switching between modes (e.g. list → create → delete).

### Scenario D — Uniform background (HackerTheme)
1. Set theme to `hacker` in the profile.
2. Open the TUI in a terminal that is wider/taller than the content.
3. Observe that the entire terminal background is uniformly `#0D0208` (dark
   green-black), with no areas showing the terminal emulator's default colour.
4. Resize the terminal window — background remains uniform after resize.

### Scenario E — Auth unsupported key error
1. Configure a profile with an SSH key type not supported for encryption.
2. Navigate to **Projects** to trigger authentication.
3. Observe that an error modal appears explaining the failure instead of
   silently redirecting to the main menu.

### Scenario F — Profiles split layout
1. Open the TUI with at least two profiles configured.
2. Navigate to **Profiles**.
3. Observe a two-pane layout: left side shows the list of profile names, right
   side shows the details (Name, Server URL, Identity file, Theme) of the
   highlighted profile.
4. Press `↓` to move to the next profile — observe the right pane updates
   immediately without any delay.
5. The active profile is visually marked with `*` in the left pane.
6. Press `e` to enter edit mode — observe the split layout disappears and the
   full-width form is shown; press `Esc` to return to the split list.

### Scenario G — Secrets tabular display
1. Navigate to a project that has at least two secrets.
2. Observe that secrets are displayed in a table with:
   - A header row (`KEY │ VALUE`).
   - A horizontal separator line beneath the header.
   - Each row aligned: keys padded to the same width, values in the second column.
3. Move the cursor up/down — the selected row is highlighted.
4. Press `n` to add a secret — the inline add form appears (unchanged from before).

---

## Acceptance criteria

- All automated tests pass: `go build ./...` ✓, `go test ./...` ✓.
- All component checklists above are ticked.
- Manual scenarios A–G pass without visual regressions on the other screens.
- No leftover dead code from the old inline error/loading/hints approach.




