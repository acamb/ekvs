# validation.md — tui_e2e_tests

Describes how to verify that the `tui_e2e_tests` feature is correctly
implemented.

---

## Automated checks (must all pass)

```bash
# Full suite — includes all existing unit tests and new e2e tests.
go build ./...
go test ./...

# Target only e2e tests.
go test ./internal/tui/... -run TestE2E -v
```

---

## E2e test inventory

Run `go test ./internal/tui/... -run TestE2E -v 2>&1 | grep "^=== RUN"` to confirm
all 34 e2e functions are registered. Expected output (one line per test):

```
=== RUN   TestE2E_Projects_LoadRendersProjectList
=== RUN   TestE2E_Projects_NavigateChangesHighlight
=== RUN   TestE2E_Projects_CreateProjectFlow
=== RUN   TestE2E_Projects_DeleteProjectFlow
=== RUN   TestE2E_Projects_ErrorModalDismissFlow
=== RUN   TestE2E_Projects_FooterHintsInAllModes
=== RUN   TestE2E_Projects_SpinnerVisibleDuringLoad
=== RUN   TestE2E_Secrets_LoadRendersTable
=== RUN   TestE2E_Secrets_AddSecretFlow
=== RUN   TestE2E_Secrets_DeleteSecretFlow
=== RUN   TestE2E_Secrets_SearchAndClearCycle
=== RUN   TestE2E_Secrets_ErrorModalFlow
=== RUN   TestE2E_Secrets_FooterHintsListMode
=== RUN   TestE2E_Profiles_SplitLayoutWithNavigation
=== RUN   TestE2E_Profiles_ActiveProfileMarked
=== RUN   TestE2E_Profiles_EditModeHidesSplitLayout
=== RUN   TestE2E_Profiles_DeleteProfileFlow
=== RUN   TestE2E_Profiles_CreateModeEntersForm
=== RUN   TestE2E_Profiles_FooterHintsListMode
=== RUN   TestE2E_Profiles_CursorWraps
=== RUN   TestE2E_Auth_ViewShowsPassphrasePrompt
=== RUN   TestE2E_Auth_UnencryptedKeySucceedsWithoutPassphrase
=== RUN   TestE2E_Auth_WrongThenCorrectPassphrase
=== RUN   TestE2E_Auth_EscCancels
=== RUN   TestE2E_Auth_InputMasked
=== RUN   TestE2E_Wizard_InitialViewShowsNamePrompt
=== RUN   TestE2E_Wizard_FullCompletionFlowNoSave
=== RUN   TestE2E_Wizard_PickListVisibleWhenKeysFound
=== RUN   TestE2E_Wizard_PickListNavigationUpdatesView
=== RUN   TestE2E_Wizard_SwitchToManualAndBackFlow
=== RUN   TestE2E_Wizard_EmptyNameSkipsAdvance
=== RUN   TestE2E_Root_MainMenuContents
=== RUN   TestE2E_Root_MenuNavigationChangesHighlight
=== RUN   TestE2E_Root_ProfileSelectShowsMultipleProfiles
=== RUN   TestE2E_Root_WindowResizeUpdatesModelAndView
=== RUN   TestE2E_Root_FooterVisibleInMainMenu
=== RUN   TestE2E_Root_WizardShownWhenNoProfiles
=== RUN   TestE2E_Root_SingleProfileSkipsSelect
=== RUN   TestE2E_Root_AuthenticatedSessionNavigatesToProjects
```

---

## Per-package checklists

### `projects`

- [ ] P-1: view contains "alpha", "beta", "gamma" and "> alpha" after FetchedMsg.
- [ ] P-2: after ↓ view shows "> second"; after ↑ shows "> first".
- [ ] P-3: `fc.createCalled == "newproj"` after create flow; "newproj" in final list.
- [ ] P-4: `fc.deleteCalled == "to-delete"` after delete flow; "to-delete" absent; "keep" present.
- [ ] P-5: view contains "Error" and error message; after dismiss mode is `modeList` and message gone.
- [ ] P-6: list footer has "n"/"d"; create footer has "confirm"/"cancel"; delete footer has "y"/"n".
- [ ] P-7: loading view contains "Loading".

### `secrets`

- [ ] S-1: view contains "KEY", "VALUE", "API_KEY", "DB_PASS".
- [ ] S-2: `fc.setCalled[0] == "MYKEY"`; encrypted value non-empty; FetchedMsg returned.
- [ ] S-3: `fc.deleteCalled == "OLD_KEY"`; FetchedMsg returned.
- [ ] S-4: after `/` + "apple" → BANANA_KEY absent; Esc keeps filter; second Esc clears.
- [ ] S-5: "Error" + "server down" visible; after dismiss mode is `modeList`.
- [ ] S-6: list footer contains "n", "d", "/".

### `profiles`

- [ ] PR-1: after WindowSizeMsg `width=120` → view contains both profile names and first URL in detail.
      After ↓ → second URL in detail pane.
- [ ] PR-2: `*` visible in view for active profile.
- [ ] PR-3: after `e` → `mode == modeEdit`; "Profile" label visible.
- [ ] PR-4: after ↓ → `d` → `y` → `ConfigChangedMsg` emitted.
- [ ] PR-5: after `c` → `mode == modeCreate`.
- [ ] PR-6: list footer contains "c", "e", "d".
- [ ] PR-7: after two ↓ presses → `cursor == 0`.

### `auth`

- [ ] A-1: initial view contains "passphrase" (case-insensitive).
- [ ] A-2: Init → tryLoadMsg → AuthSuccessMsg for unencrypted key.
- [ ] A-3: wrong pass → `showModal == true`; DismissMsg → `showModal == false`; correct pass → AuthSuccessMsg.
- [ ] A-4: Esc → AuthCancelMsg.
- [ ] A-5: view does not contain "secret" (the typed text); view contains "******".

### `wizard`

- [ ] W-1: initial view contains "Profile".
- [ ] W-2: all steps complete → DoneMsg with `Profile.Name == "myprod"` and `Base(IdentityFile) == "id_ed25519"`.
- [ ] W-3: pick list view contains both key names from fakeSSHDir.
- [ ] W-4: after ↓, `selectedIdentity()` returns second key.
- [ ] W-5: `m` → `identMode == identityModeManual`; Esc → `identMode == identityModePick`; pick list in view.
- [ ] W-6: Enter on empty name → `step == stepName` unchanged.

### `root`

- [ ] R-1: view contains "Projects", "Profiles", "Quit".
- [ ] R-2: after ↓ `cursor == 1`; after ↑ `cursor == 0`.
- [ ] R-3: `screen == screenProfileSelect`; view contains "production" and "staging".
- [ ] R-4: `width == 200`, `height == 50`; view non-empty.
- [ ] R-5: view contains "navigate" and "quit".
- [ ] R-6: `screen == screenWizard`; view contains "Profile".
- [ ] R-7: `screen == screenMain`; view contains "Projects".
- [ ] R-8: `screen == screenProjects`; view contains "Projects".

---

## Acceptance criteria

- `go build ./...` ✓
- `go test ./...` ✓ (no regressions in existing tests)
- All 39 `TestE2E_*` functions pass.
- No e2e test accesses a real server, `~/.ssh`, or writes to disk outside `t.TempDir()`.
- No new `go.mod` dependencies.
