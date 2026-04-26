# validation.md — tui_profiles

## Acceptance criteria

The feature is considered complete only when all of the following are true:
- the TUI exposes a profile-management screen reachable from the main menu;
- profile create, edit, and delete operations are persisted to the YAML config file;
- switching to a different profile clears the current session state before the new profile becomes active;
- deleting the active profile redirects correctly according to the roadmap;
- all relevant unit tests pass with no regressions.

---

## 1. Unit tests

**Linux / macOS**:
```bash
make test
```

At minimum, the following targeted command must also pass:

```bash
go test ./internal/tui/... -v -cover
```

**Windows** (PowerShell or cmd):
```cmd
go test ./internal/tui/... -v -cover
go test ./... -count=1
```

### Required tests — `internal/tui/config`

| Test name | What it verifies |
|-----------|------------------|
| `TestConfig_ExpandHome_TildeReplaced` | `ExpandHome` replaces `~` with the user home directory |
| `TestConfig_ExpandHome_AbsolutePathUnchanged` | `ExpandHome` leaves absolute paths unchanged |
| `TestConfig_SSHDir_ReturnsValidPath` | `SSHDir` returns a non-empty path using `os.UserHomeDir()` |
| `TestConfig_UpsertProfile_AddsNew` | adding a profile with a unique name succeeds |
| `TestConfig_UpsertProfile_ReplacesExisting` | upserting a profile whose name already exists replaces the entry in-place |
| `TestConfig_UpsertProfile_RejectsEmptyName` | `UpsertProfile` returns an error when `Profile.Name` is empty |
| `TestConfig_UpdateProfile_RenameSuccess` | renaming a profile to a new unique name succeeds |
| `TestConfig_UpdateProfile_RenameConflict` | renaming a profile to an existing name is rejected |
| `TestConfig_UpdateProfile_NotFound` | calling `UpdateProfile` with an unknown `oldName` returns an error |
| `TestConfig_DeleteProfile_Success` | deleting an existing profile removes it from the config |
| `TestConfig_DeleteProfile_NotFound` | deleting a missing profile returns an error |
| `TestConfig_SaveLoadRoundTrip_AfterMutations` | create/edit/delete survives YAML save-load round-trip |

### Required tests — `internal/tui/profiles`

| Test name | What it verifies |
|-----------|------------------|
| `TestProfilesModel_ListNavigation` | cursor moves through the profile list and wraps correctly |
| `TestProfilesModel_ViewMarksActiveProfile` | the active profile is visibly marked in `View()` |
| `TestProfilesModel_CreateFlow_DiscoveredKeysListed` | create mode shows key candidates discovered via `config.SSHDir()` (mocked) |
| `TestProfilesModel_CreateFlow_ManualIdentityPathAvailable` | create mode always allows manual `identity_file` entry |
| `TestProfilesModel_CreateFlow_NoDiscoveredKeysStillWorks` | profile creation still works when `config.SSHDir()` yields no keys |
| `TestProfilesModel_CreateFlow_Submit` | valid create form emits `profiles.ConfigChangedMsg` carrying the new profile |
| `TestProfilesModel_CreateFlow_DuplicateNameError` | duplicate names are rejected in the UI flow |
| `TestProfilesModel_EditFlow_SelectedProfile` | edit mode is available for the selected profile, whether active or not |
| `TestProfilesModel_EditFlow_RenameSubmit` | editing the selected profile can change `name` and emits the updated config |
| `TestProfilesModel_EditFlow_RenameConflict` | renaming to an already used name is rejected |
| `TestProfilesModel_ActiveProfileEdit_PromptsReload` | saving any change to the active profile shows the reload confirmation prompt |
| `TestProfilesModel_ActiveProfileEdit_ConfirmReload_EmitsReloadMsg` | confirming reload emits `profiles.ReloadActiveMsg` |
| `TestProfilesModel_ActiveProfileEdit_DeclineReload_KeepsListView` | declining reload returns to the list view without emitting a switch |
| `TestProfilesModel_InactiveProfileEdit_NoReloadPrompt` | saving changes to a non-active profile does not show the reload prompt |
| `TestProfilesModel_DeleteConfirm` | confirming deletion emits `profiles.ConfigChangedMsg` without the deleted profile |
| `TestProfilesModel_BackOnEsc` | `Esc` emits `profiles.BackMsg` |

### Required tests — `internal/tui/root`

| Test name | What it verifies |
|-----------|------------------|
| `TestRoot_OpenProfilesScreen` | selecting the menu entry opens the profile-management screen |
| `TestRoot_MainMenuReplacesSettingsWithProfiles` | the main menu exposes `Profiles` and no longer shows `Settings` |
| `TestRoot_ProfileSwitch_ClearsSession` | handling `profiles.SwitchMsg` clears signer, fingerprint, and encryption key |
| `TestRoot_ProfileSwitch_ReturnsToMain` | switching profile returns to the main menu with the new active profile |
| `TestRoot_MultipleProfilesStartupRequiresSelection` | on startup with multiple profiles, the app shows profile selection instead of remembering the last runtime choice |
| `TestRoot_ActiveProfileEdit_ReloadAccepted_SwitchesProfile` | accepting the reload prompt (`profiles.ReloadActiveMsg`) clears the current session and performs a switch to the just-updated profile |
| `TestRoot_ActiveProfileEdit_ReloadDeclined_KeepsRuntimeSession` | declining the reload prompt keeps the current runtime session/profile active |
| `TestRoot_InactiveProfileEdit_KeepsRuntimeSession` | editing a non-active profile does not change the current runtime session or active profile |
| `TestRoot_DeleteActiveProfile_WithRemainingProfiles_GoesToProfileSelect` | active-profile deletion with remaining profiles redirects to selection |
| `TestRoot_DeleteActiveProfile_LastProfile_GoesToWizard` | deleting the last active profile redirects to the first-run wizard |
| `TestRoot_SaveFailure_KeepsCurrentState` | persistence errors do not corrupt the active in-memory state |

---

## 2. Build

**Linux / macOS**:
```bash
go build ./cmd/tui/...
```

**Windows** (PowerShell or cmd):
```cmd
go build ./cmd/tui/...
```

Must complete without errors on both platforms.

---

## 3. Smoke test — create a second profile

Start from a config file containing a single profile.

**Linux / macOS**:
```bash
./tui --config ekvs-tui.yaml
```

**Windows**:
```cmd
tui.exe --config ekvs-tui.yaml
```

Steps:
1. Open the `Profiles` screen from the main menu.
2. Press `c`.
3. Verify that the `identity_file` step proposes keys discovered from the
   platform SSH directory (e.g. `~/.ssh/` on Linux, `%USERPROFILE%\.ssh\` on
   Windows) when available.
4. Enter a new unique profile name.
5. Choose one discovered key or enter a custom `identity_file` path manually.
6. Save the profile.

Expected result:
- the new profile appears in the list;
- the original profile remains active;
- the YAML file now contains two profiles.

Manual-entry check:
- even when discovered keys are listed, the UI still allows typing a custom path.

---

## 4. Smoke test — create profile when platform SSH directory yields no keys

Precondition:
- run the TUI in an environment where `config.SSHDir()` is empty, missing, or
  contains no discoverable private keys.

Steps:
1. Open `Profiles`.
2. Press `c`.
3. Proceed to the `identity_file` step.
4. Enter a custom path manually.
5. Save the profile.

Expected result:
- profile creation does not fail just because discovery found no keys;
- no discovered-key selection is required;
- the manually entered path is persisted in YAML.

---

## 5. Smoke test — edit a profile (non-active)

From the profiles screen:
1. Select a profile that is not active.
2. Press `e`.
3. Change `server_url`, `identity_file`, or `theme`.
4. Save.

Expected result:
- the list refreshes with the updated values;
- the YAML file contains the updated fields;
- no reload prompt appears;
- the current runtime theme, session, and active profile are unchanged.

---

## 6. Smoke test — edit the active profile

From the profiles screen:
1. Select the active profile.
2. Press `e`.
3. Change any field (`name`, `server_url`, `identity_file`, or `theme`).
4. Save.

Expected result:
- the TUI shows a `Reload profile now? [y/N]` prompt after every successful save
  of the active profile, regardless of which fields were changed;
- answering `yes` performs a runtime switch to the just-edited profile, clears
  the current session, and forces a fresh auth flow on the next authenticated
  action;
- answering `no` keeps the current runtime session/profile active until the user
  explicitly reloads or switches later;
- in both cases, the YAML file contains the updated profile data.

Rename check:
- change the active profile `name` to a new unique value and save;
- the list shows the new name and the reload prompt still appears;
- attempting to rename it to a duplicate name shows a validation error and does
  not save.

---

## 7. Smoke test — edit a non-active profile

From the profiles screen:
1. Select a profile that is not active.
2. Press `e`.
3. Change any field (`name`, `server_url`, `identity_file`, or `theme`).
4. Save.

Expected result:
- no reload prompt is shown;
- the YAML file contains the updated profile data;
- the active profile, runtime theme, and current session remain unchanged.

---

## 8. Smoke test — switch profile resets auth state

Precondition:
- authenticate with profile `A` by entering the Projects or Secrets flow once;
- confirm that the session has been established.

Then:
1. Return to the main menu.
2. Open `Profiles`.
3. Select profile `B`.
4. Press `Enter` (or the final switch key decided during implementation).

Expected result:
- the app returns to the main menu with profile `B` active;
- the previous session is no longer authenticated;
- opening `Projects` or `Secrets` triggers the auth flow again instead of reusing profile `A`'s credentials.

---

## 9. Smoke test — restart with multiple profiles still asks selection

Precondition:
- the config file contains at least two profiles;
- during the previous run you switched to one of them.

Steps:
1. Quit the TUI.
2. Start it again with the same config file.

Expected result:
- the TUI shows the profile-selection screen at startup;
- it does not silently reuse the profile selected in the previous runtime session.

---

## 10. Smoke test — delete a non-active profile

From the profiles screen:
1. Select a profile that is not active.
2. Press `d`.
3. Confirm deletion.

Expected result:
- the selected profile disappears from the list;
- the active profile marker stays on the original active profile;
- the YAML file no longer contains the deleted profile.

---

## 11. Smoke test — delete the active profile with remaining profiles

Precondition:
- the config file contains at least two profiles.

Steps:
1. Open `Profiles`.
2. Select the active profile.
3. Press `d` and confirm.

Expected result:
- the app redirects to the profile-selection screen;
- the deleted profile is absent from the YAML file;
- selecting another profile returns to the main menu;
- the next authenticated action triggers a fresh auth flow.

---

## 12. Smoke test — delete the last remaining active profile

Precondition:
- the config file contains exactly one profile.

Steps:
1. Open `Profiles`.
2. Delete the only active profile.

Expected result:
- the app redirects to the first-run wizard;
- the user is guided to create a replacement profile;
- once the flow completes, the YAML file contains the replacement profile and the app returns to the main menu.

---

## 13. Regression checks

After implementation, verify that the existing TUI flows still work:
- startup with no config file still opens the wizard;
- startup with multiple profiles still opens the profile-selection screen;
- the main menu shows `Profiles` instead of `Settings`;
- profile creation still works both with discovered keys and with manual identity-file entry;
- profile creation works when `config.SSHDir()` fails or yields no results;
- `ExpandHome` is called for the default `IdentityFile` and the resulting path is absolute;
- auth flow still works for an unchanged active profile;
- projects and secrets flows still use the active profile's `server_url`;
- `go build ./cmd/tui/...` succeeds on both Linux and Windows.






