# plan.md — tui_e2e_tests

Ordered list of atomic implementation tasks. Each task leaves
`go build ./...` and `go test ./...` passing.

---

## Task 1 — Evaluate `teatest` dependency compatibility

**Description**
Run `go get github.com/charmbracelet/x/exp/teatest` and inspect the module's
import graph. **Finding**: `teatest` depends on
`github.com/charmbracelet/bubbletea` (v1), incompatible with the project's
`charm.land/bubbletea/v2`. Remove the dependency (`go get
github.com/charmbracelet/x/exp/teatest@none && go mod tidy`) and proceed with
direct model-driving tests.

**Files**
- `go.mod` / `go.sum` (no net change — dependency never committed)

---

## Task 2 — Create `internal/tui/projects/projects_e2e_test.go`

**Description**
Add `TestE2E_Projects_*` tests in `package projects`:
- `TestE2E_Projects_LoadRendersProjectList` — fetch → view contains all names
- `TestE2E_Projects_NavigateChangesHighlight` — ↓/↑ moves cursor in view
- `TestE2E_Projects_CreateProjectFlow` — n → type → Enter → CreateProject called → name in list
- `TestE2E_Projects_DeleteProjectFlow` — d → y → DeleteProject called → name gone
- `TestE2E_Projects_ErrorModalDismissFlow` — ErrMsg → modal → Enter → back to list
- `TestE2E_Projects_FooterHintsInAllModes` — list/create/delete footer content
- `TestE2E_Projects_SpinnerVisibleDuringLoad` — loading view contains "Loading"

Reuses helpers from `projects_test.go` (`fakeClient`, `keyMsg`, `sendKey`,
`applyFetched`, `newTestModel`, `newWithClient`). Adds local `e2eTheme` helper.

**Files**
- `internal/tui/projects/projects_e2e_test.go`

---

## Task 3 — Create `internal/tui/secrets/secrets_e2e_test.go`

**Description**
Add `TestE2E_Secrets_*` tests in `package secrets`:
- `TestE2E_Secrets_LoadRendersTable` — KEY/VALUE header + key names in view
- `TestE2E_Secrets_AddSecretFlow` — n → key → Enter → value → Enter → SetSecret called
- `TestE2E_Secrets_DeleteSecretFlow` — d → y → DeleteSecret called
- `TestE2E_Secrets_SearchAndClearCycle` — / → query → filter → Esc (keep) → Esc (clear)
- `TestE2E_Secrets_ErrorModalFlow` — ErrMsg → modal → dismiss
- `TestE2E_Secrets_FooterHintsListMode` — n, d, / in footer

Reuses `fakeClient`, `sendKey`, `applyFetched`, `testSession`, `encryptBlob`
from `secrets_test.go`. Adds local `e2eSecretsTheme`.

**Files**
- `internal/tui/secrets/secrets_e2e_test.go`

---

## Task 4 — Create `internal/tui/profiles/profiles_e2e_test.go`

**Description**
Add `TestE2E_Profiles_*` tests in `package profiles`:
- `TestE2E_Profiles_SplitLayoutWithNavigation` — WindowSizeMsg + ↓ updates detail pane
- `TestE2E_Profiles_ActiveProfileMarked` — active profile has `*` in view
- `TestE2E_Profiles_EditModeHidesSplitLayout` — e → modeEdit → form visible
- `TestE2E_Profiles_DeleteProfileFlow` — d → y → ConfigChangedMsg emitted
- `TestE2E_Profiles_CreateModeEntersForm` — c → modeCreate
- `TestE2E_Profiles_FooterHintsListMode` — c, e, d in footer
- `TestE2E_Profiles_CursorWraps` — cursor wraps at end of list

Reuses `testConfig`, `newTestModel`, `sendKey`, `makeTempConfigPath` from
`profiles_test.go`.

**Files**
- `internal/tui/profiles/profiles_e2e_test.go`

---

## Task 5 — Create `internal/tui/auth/auth_e2e_test.go`

**Description**
Add `TestE2E_Auth_*` tests in `package auth`:
- `TestE2E_Auth_ViewShowsPassphrasePrompt` — initial view contains "passphrase"
- `TestE2E_Auth_UnencryptedKeySucceedsWithoutPassphrase` — Init → success
- `TestE2E_Auth_WrongThenCorrectPassphrase` — complete retry journey
- `TestE2E_Auth_EscCancels` — Esc → AuthCancelMsg
- `TestE2E_Auth_InputMasked` — view shows `******`

Reuses `newModel`, `runCmd` from `auth_test.go`. Uses testdata keys.

**Files**
- `internal/tui/auth/auth_e2e_test.go`

---

## Task 6 — Create `internal/tui/wizard/wizard_e2e_test.go`

**Description**
Add `TestE2E_Wizard_*` tests in `package wizard`:
- `TestE2E_Wizard_InitialViewShowsNamePrompt` — view contains "Profile"
- `TestE2E_Wizard_FullCompletionFlowNoSave` — all steps → DoneMsg with correct fields
- `TestE2E_Wizard_PickListVisibleWhenKeysFound` — both key names in pick-list view
- `TestE2E_Wizard_PickListNavigationUpdatesView` — ↓ changes selectedIdentity
- `TestE2E_Wizard_SwitchToManualAndBackFlow` — m → manual → Esc → pick
- `TestE2E_Wizard_EmptyNameSkipsAdvance` — Enter on empty stays on stepName

Reuses `testTheme`, `fakeSSHDir`, `press`, `keyMsg`, `advanceToIdentityStep`
from `wizard_test.go`.

**Files**
- `internal/tui/wizard/wizard_e2e_test.go`

---

## Task 7 — Create `internal/tui/root/root_e2e_test.go`

**Description**
Add `TestE2E_Root_*` tests in `package root`:
- `TestE2E_Root_MainMenuContents` — view contains Projects/Profiles/Quit
- `TestE2E_Root_MenuNavigationChangesHighlight` — ↓/↑ moves cursor
- `TestE2E_Root_ProfileSelectShowsMultipleProfiles` — both profiles in view
- `TestE2E_Root_WindowResizeUpdatesModelAndView` — width/height updated
- `TestE2E_Root_FooterVisibleInMainMenu` — "navigate"/"quit" in footer
- `TestE2E_Root_WizardShownWhenNoProfiles` — screenWizard; "Profile" in view
- `TestE2E_Root_SingleProfileSkipsSelect` — screenMain; "Projects" in view
- `TestE2E_Root_AuthenticatedSessionNavigatesToProjects` — triggerProjectsMsg → screenProjects

Reuses `newRootWithProfile`, `newRootAuthenticated` from `root_test.go`.

**Files**
- `internal/tui/root/root_e2e_test.go`

---

## Task 8 — Final verification

**Description**
- Run `go build ./...` — must succeed.
- Run `go test ./...` — all tests pass, no regressions.
- Run `go test ./internal/tui/... -run TestE2E -v` — all e2e tests listed and pass.
- Confirm no test accesses a real server, `~/.ssh`, or writes to the filesystem
  outside `t.TempDir()`.

**Files**
- No changes; verification only.
