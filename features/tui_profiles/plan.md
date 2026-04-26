# plan.md — tui_profiles

## Ordered task list

---

### 1. Extend the TUI configuration model for runtime profile management
Update `internal/tui/config/config.go` so the TUI can safely mutate the profile list and persist it back to YAML.

#### 1a. Platform-aware path helpers (new, required on both Linux and Windows)
Add these two exported helpers:

```go
// ExpandHome replaces a leading "~" with os.UserHomeDir().
// Returns the path unchanged if it does not start with "~".
func ExpandHome(path string) string

// SSHDir returns <home>/.ssh on Linux/macOS and <home>\.ssh on Windows.
// Returns "", err if os.UserHomeDir() fails.
func SSHDir() (string, error)
```

Update `DefaultProfile()` and `applyDefaults` to call `ExpandHome` so the default
`IdentityFile` value is always an absolute path.

#### 1b. Profile CRUD helpers
Add the following mutation helpers to `ConfigFile`:

- `UpsertProfile(profile Profile) error` — true upsert: adds the profile if its
  name is new, replaces the existing entry in-place if the name is already
  present. Returns an error only when `profile.Name` is empty.
- `UpdateProfile(oldName string, profile Profile) error` — rename-safe update:
  locates the entry by `oldName`, validates uniqueness of the new name (if it
  changed), then replaces. Returns an error if `oldName` is not found or if the
  new name conflicts with a different profile.
- `DeleteProfile(name string) error` — removes the profile with the given name;
  returns an error if not found.
- `FindProfile(name string) (Profile, int, bool)` — index-based lookup helper.

Keep `Save(path, cf)` as the single persistence boundary for all CRUD actions.

Notes:
- the YAML structure must stay compatible with the current `profiles:` list;
- profile switching is a runtime action and does **not** add a new `active_profile` field;
- when more than one profile exists, startup must continue to show the profile selection screen every time.

---

### 2. Expand `internal/tui/config` unit tests
Update `internal/tui/config/config_test.go` with table-driven tests for the new profile-management helpers.

Required scenarios:
- `ExpandHome` replaces `~` with the user home directory on the current platform;
- `ExpandHome` leaves absolute paths unchanged;
- `SSHDir` returns a valid path on the current platform;
- `UpsertProfile` adds a new unique profile;
- `UpsertProfile` replaces an existing profile in-place when the name matches;
- `UpsertProfile` returns an error for an empty profile name;
- `UpdateProfile` renames a profile to a new unique name;
- `UpdateProfile` returns an error when the new name conflicts with another profile;
- `UpdateProfile` returns an error when oldName is not found;
- `DeleteProfile` removes an existing profile;
- `DeleteProfile` returns an error when the name is not found;
- save/load round-trip after create, edit, and delete operations preserves all fields.

---

### 3. Add a dedicated profile-management screen
Create `internal/tui/profiles/profiles.go` (and tests) as the Bubble Tea model for profile CRUD and switching.

The screen should support the following modes:
- `modeList` — browse profiles and see which one is active;
- `modeCreate` — create a new profile;
- `modeEdit` — edit the selected profile;
- `modeDeleteConfirm` — confirm deletion.

Minimum data shown in list mode:
- profile `name`;
- `server_url`;
- `identity_file`;
- `theme`;
- active marker for the current profile.

Draft keyboard map:
- `↑`/`↓`, `j`/`k`: move cursor;
- `c`: create profile;
- `e`: edit the selected profile;
- `s` or `Enter`: switch to the selected profile;
- `d`: delete the selected profile;
- `esc`: go back to the main menu;
- `q`/`Ctrl+C`: quit the application.

The create/edit form should collect:
- `name`;
- `server_url`;
- `identity_file`;
- `theme`.

In create mode, the `identity_file` step must:
- call `config.SSHDir()` to obtain the platform-appropriate SSH directory
  (works on both Linux and Windows);
- scan that directory for candidate private-key files;
- present the discovered keys as selectable options;
- always allow entering a custom path manually.

If the active profile is edited and saved, the flow **always** shows a
confirmation prompt asking whether to reload the profile now, regardless of
which fields changed. Editing a non-active profile must not show this prompt.

---

### 4. Define profile-management messages and outcomes
Add the messages needed to communicate from the profile screen back to `internal/tui/root`.

Following the convention used by `projects` and `secrets`, the profile screen
exports its own message types from `internal/tui/profiles/messages.go`:

| Type | Payload | Purpose |
|------|---------|---------|
| `profiles.BackMsg` | — | return to main menu (Esc) |
| `profiles.SwitchMsg` | `Profile tuiconfig.Profile` | switch to selected profile |
| `profiles.ReloadActiveMsg` | `Profile tuiconfig.Profile` | user confirmed reload of active profile after edit |
| `profiles.ConfigChangedMsg` | `Config *tuiconfig.ConfigFile` | CRUD succeeded; root must update its in-memory config |

Root defines the unexported trigger in `internal/tui/root/messages.go`:

```go
// triggerProfilesMsg requests navigation to the Profiles screen from the main menu.
type triggerProfilesMsg struct{}
```

The root model remains responsible for:
- updating the active profile;
- reloading the theme;
- clearing session state on switch or confirmed reload;
- deciding which screen comes next after deletion of the active profile.

`profiles.SwitchMsg` and `profiles.ReloadActiveMsg` are distinct types so
root can log/trace the origin, but both trigger the same runtime sequence:
`session.Clear()` → set new active profile → resolve theme → show main menu.

The profile screen remains responsible for local form state such as:
- discovered key candidates from `config.SSHDir()`;
- selection vs manual-entry mode for `identity_file`.

---

### 5. Integrate the profile screen into the root model
Update `internal/tui/root/root.go`, `internal/tui/root/menu.go`, and `internal/tui/root/messages.go`.

Planned root changes:
- add `screenProfiles`;
- store the loaded `*tuiconfig.ConfigFile` inside the root model so it can be mutated in-memory;
- store the config file path in the root model so all CRUD changes can be persisted;
- add a `profiles.Model` sub-model;
- replace the existing `Settings` menu entry with `Profiles` as the single entry point for connection-profile management.

The main menu must expose `Profiles` instead of `Settings`.

---

### 6. Persist profile CRUD actions to YAML
Every create, edit, and delete action must write the updated configuration back to the YAML file passed via `--config` (or the default `ekvs-tui.yaml`).

Rules:
- when the app was started without an existing config file, the first profile-management mutation creates the file at the configured path;
- failed saves must keep the user on the profile screen and display the error;
- in-memory state must only be considered committed after a successful save.

---

### 7. Implement switch semantics and session reset
When switching from the current profile to a different one:
- call `session.Clear()` before loading the new profile into the root model;
- replace `profile`, `theme`, and any screen-specific clients derived from the old profile;
- return to the main menu;
- do **not** trigger authentication immediately;
- the next authenticated action must follow the existing lazy auth flow from `tui_auth`.

Additional test expectation:
- after a switch, any previously loaded signer/fingerprint/encryption key is gone.
- after quitting and restarting with multiple profiles, the app asks again which profile to use.

---

### 8. Handle edits to the active profile
After saving any change to the active profile (regardless of which fields
changed):
- persist the updated configuration first;
- show a confirmation prompt `Reload profile now? [y/N]`;
- if the user confirms, emit `profiles.ReloadActiveMsg{Profile: updatedProfile}`;
  root handles this identically to `profiles.SwitchMsg` (session cleared, theme
  re-resolved, main menu shown with the updated profile active);
- if the user declines, keep the current runtime profile/session and return to
  the list view.

When saving changes to a non-active profile:
- persist the updated configuration;
- do not show a reload prompt;
- do not switch the active profile;
- do not change the current runtime session;
- emit `profiles.ConfigChangedMsg` and return to the list view.

---

### 9. Handle deletion of the active profile
Implement the roadmap rule exactly:
- deleting a non-active profile keeps the current active profile unchanged;
- deleting the active profile when other profiles remain redirects to the
  profile-selection screen;
- deleting the active profile when no profiles remain redirects to the existing
  first-run wizard.

For the last case, the wizard's existing optional-save behaviour is **preserved
unchanged**: the user decides whether to save and where. No new mandatory-save
mode is introduced. After the wizard completes (via `wizard.DoneMsg`), root
transitions to the main menu as in any other first-run flow.

---

### 10. Add unit tests for `internal/tui/profiles` and `internal/tui/root`
Required test coverage:
- opening the profiles screen from the main menu;
- presence of `Profiles` in the main menu and absence of `Settings`;
- list navigation and active-profile marker;
- create flow persists a new profile and updates the in-memory config;
- create flow proposes discovered keys using `config.SSHDir()` (mocked);
- create flow allows manual `identity_file` entry even when discovered keys exist;
- create flow still works when `config.SSHDir()` returns empty or an error;
- edit flow updates the selected profile, including rename;
- rename conflicts are rejected;
- switch flow emits `profiles.SwitchMsg` and root clears session state;
- restarting with multiple profiles reopens the profile-selection screen;
- saving any change to the active profile shows the reload prompt;
- confirming reload (`profiles.ReloadActiveMsg`) clears session and behaves like a switch;
- declining reload keeps the current runtime session unchanged;
- saving any change to a non-active profile does not show a reload prompt;
- deleting a non-active profile keeps the active profile;
- deleting the active profile with remaining profiles opens profile selection;
- deleting the last profile opens the wizard;
- save failures are rendered as errors and do not corrupt the in-memory state.

---

### 11. Update the documentation example and verify the build
Update `ekvs-tui.yaml.example` only if the final spec introduces new YAML fields.

Validation commands — **Linux / macOS**:
```bash
go test ./internal/tui/... -v -cover
go build ./cmd/tui/...
make test
```

Validation commands — **Windows** (PowerShell or cmd):
```cmd
go test ./internal/tui/... -v -cover
go build ./cmd/tui/...
go test ./... -count=1
```

All commands must complete without regressions on both platforms.





