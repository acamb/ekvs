# requirements.md — tui_e2e_tests

## Goal

Add end-to-end tests for every TUI screen. Each screen gets a `*_e2e_test.go`
file that drives the model through realistic multi-step interaction sequences
via simulated key presses and asserts on the rendered `View().Content` output,
without requiring a real server or real SSH keys.

---

## Scope

### In scope
- One `*_e2e_test.go` per screen package:
  - `internal/tui/projects/`
  - `internal/tui/secrets/`
  - `internal/tui/profiles/`
  - `internal/tui/auth/`
  - `internal/tui/wizard/`
  - `internal/tui/root/` (covers `mainModel`, `profileSelect`, and the root
    dispatch loop)
- Each file lives in the **same package** as the model (e.g. `package projects`)
  so it can access unexported helpers already defined in the corresponding
  `*_test.go` unit-test file (`fakeClient`, `keyMsg`, `sendKey`, etc.).
- Assertions are inline (`strings.Contains`); no golden files.
- No new production dependencies; no new test dependencies.
- Both **happy paths** and **error paths** are covered (see FR per screen).

### Out of scope
- Real HTTP calls (the fake client replaces all network I/O).
- Real SSH keys on disk (test key files under `internal/ssh/testdata/` are
  used; no access to `~/.ssh`).
- CLI or server packages.
- New functional features.

---

## Functional Requirements

### FR-1 projects screen

File: `internal/tui/projects/projects_e2e_test.go`

| # | Scenario | Simulated input | Expected output |
|---|----------|----------------|-----------------|
| P-1 | Load renders project list | `FetchedMsg{["alpha","beta","gamma"]}` | view contains all names; first item highlighted |
| P-2 | Navigate changes highlight | ↓ then ↑ | cursor moves to second item then back to first |
| P-3 | Create project flow | `n` → type name → `Enter` | `CreateProject` called; `FetchedMsg` returns; name in list |
| P-4 | Delete project flow | `d` → `y` | `DeleteProject` called; name gone after `FetchedMsg` |
| P-5 | Error modal flow | `ErrMsg` → `Enter` dismiss | modal visible; dismissed; back to `modeList` |
| P-6 | Footer hints all modes | list / create / delete | `n`, `d`, `confirm`, `cancel`, `y` present |
| P-7 | Spinner visible during load | `loading = true` | view contains "Loading" |

### FR-2 secrets screen

File: `internal/tui/secrets/secrets_e2e_test.go`

| # | Scenario | Simulated input | Expected output |
|---|----------|----------------|-----------------|
| S-1 | Load renders table | `FetchedMsg` with two secrets | view contains `KEY`, `VALUE`, both key names |
| S-2 | Add secret flow | `n` → key → `Enter` → value → `Enter` | `SetSecret` called; encrypted value non-empty |
| S-3 | Delete secret flow | `d` → `y` | `DeleteSecret` called |
| S-4 | Search and clear cycle | `/` → type query → `Esc` → `Esc` | filtered view; filter kept; then cleared |
| S-5 | Error modal flow | `ErrMsg` → `Enter` | modal visible; dismissed; back to `modeList` |
| S-6 | Footer hints list mode | list mode | `n`, `d`, `/` present |

### FR-3 profiles screen

File: `internal/tui/profiles/profiles_e2e_test.go`

| # | Scenario | Simulated input | Expected output |
|---|----------|----------------|-----------------|
| PR-1 | Split layout with navigation | `WindowSizeMsg{120}` → ↓ | both profiles shown; detail pane updates |
| PR-2 | Active profile marked | one profile active | `*` in view |
| PR-3 | Edit mode hides layout | `e` | `modeEdit`; form label visible |
| PR-4 | Delete flow | ↓ → `d` → `y` | `ConfigChangedMsg` emitted |
| PR-5 | Create mode enters form | `c` | `modeCreate`; form label visible |
| PR-6 | Footer hints list mode | list mode | `c`, `e`, `d` present |
| PR-7 | Cursor wraps | ↓ past last | wraps to first |

### FR-4 auth screen

File: `internal/tui/auth/auth_e2e_test.go`

| # | Scenario | Simulated input | Expected output |
|---|----------|----------------|-----------------|
| A-1 | View shows passphrase prompt | initial view | contains "passphrase" |
| A-2 | Unencrypted key succeeds | `Init()` → `tryLoadMsg` | `AuthSuccessMsg` emitted |
| A-3 | Wrong then correct passphrase | wrong → modal → dismiss → correct | `AuthSuccessMsg` after correct |
| A-4 | Esc cancels | `Esc` | `AuthCancelMsg` emitted |
| A-5 | Input masked | type "secret" | view shows `******`, not "secret" |

### FR-5 wizard screen

File: `internal/tui/wizard/wizard_e2e_test.go`

| # | Scenario | Simulated input | Expected output |
|---|----------|----------------|-----------------|
| W-1 | Initial view shows name prompt | initial view | contains "Profile" |
| W-2 | Full completion no-save | all steps → `n` at confirm | `DoneMsg` emitted; profile name and identity file correct |
| W-3 | Pick list visible when keys found | fake SSH dir with two keys | both key names in view |
| W-4 | Pick list navigation | ↓ | `selectedIdentity` changes |
| W-5 | Switch to manual and back | `m` → `Esc` | manual mode then pick mode restored |
| W-6 | Empty name stays on step | `Enter` with empty name | `stepName` unchanged |

### FR-6 root model

File: `internal/tui/root/root_e2e_test.go`

| # | Scenario | Simulated input | Expected output |
|---|----------|----------------|-----------------|
| R-1 | Main menu contents | initial view | "Projects", "Profiles", "Quit" |
| R-2 | Menu cursor navigation | ↓ then ↑ | cursor moves to 1 then back to 0 |
| R-3 | Profile select shows both profiles | two-profile config | both names in view |
| R-4 | Window resize updates model | `WindowSizeMsg{200,50}` | `width=200`, `height=50` |
| R-5 | Footer visible in main menu | menu view | "navigate" and "quit" hints |
| R-6 | Wizard shown with no config | `New(nil,...)` | `screenWizard`; "Profile" in view |
| R-7 | Single profile skips select | one-profile config | `screenMain`; "Projects" in view |
| R-8 | Authenticated session navigates to projects | `triggerProjectsMsg` | `screenProjects`; "Projects" in view |

---

## Non-Functional Requirements

- NFR-1: `go build ./...` and `go test ./...` must pass after every task.
- NFR-2: Tests must not access the real filesystem (no `~/.ssh`, no real config
  files); all dependencies are injected via fakes or testdata files.
- NFR-3: No new external dependencies are added to `go.mod`.
- NFR-4: All test functions are named `TestE2E_*` to allow targeted runs:
  `go test ./internal/tui/... -run TestE2E`.

---

## Decisions

| # | Decision |
|---|----------|
| D-1 | `github.com/charmbracelet/x/exp/teatest` is **not used**: it depends on `github.com/charmbracelet/bubbletea` (v1), incompatible with the project's `charm.land/bubbletea/v2`. Models are driven directly via `Update()` calls. |
| D-2 | No shared `testhelper` package: each screen package already defines its own `fakeClient`, `keyMsg`, `sendKey` and session/theme helpers in the existing `*_test.go` file. The e2e file reuses them since it lives in the same package. |
| D-3 | E2e test files are in the **same package** as the model (not `_test` suffix) to access unexported symbols. |
| D-4 | Assertions are inline (`strings.Contains`, field comparisons). No golden files. |
| D-5 | Testdata SSH keys (`internal/ssh/testdata/`) are used for auth tests. The correct passphrase for `ed25519-passphrase` is `"testpass"`. |

---

## Files Created

```
internal/tui/projects/projects_e2e_test.go
internal/tui/secrets/secrets_e2e_test.go
internal/tui/profiles/profiles_e2e_test.go
internal/tui/auth/auth_e2e_test.go
internal/tui/wizard/wizard_e2e_test.go
internal/tui/root/root_e2e_test.go
```
