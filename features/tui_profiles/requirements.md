# requirements.md — tui_profiles

## User decisions

| Decision | Choice |
|----------|--------|
| Next roadmap feature | `tui_profiles` |
| Profile switching persistence | Switch is runtime-only; no `active_profile` field is written to YAML. |
| Startup with multiple profiles | The TUI must always ask which profile to use at startup when more than one profile exists. |
| Editable fields of a profile | `name`, `server_url`, `identity_file`, and `theme` are editable for any selected profile. |
| Main menu entry | Replace `Settings` with `Profiles`. |
| Create flow `identity_file` UX | Propose a list of keys auto-discovered from the platform SSH directory, while always allowing manual path entry. |
| Active profile edit (any field) | After saving any change to the active profile, prompt the user to reload it; accepting is equivalent to a full profile switch. |
| Non-active profile edit | Save the change without prompting for reload, without changing the runtime session. |
| Last-profile deletion flow | Redirect to the existing first-run wizard; the user decides whether to save and where (existing optional-save behaviour is preserved). |
| Home path resolution | `os.UserHomeDir()` is used everywhere; `~` prefixes are expanded via `config.ExpandHome(path)` and the platform SSH dir is obtained via `config.SSHDir()`. Both work on Linux and Windows. |
| `UpsertProfile` semantics | `UpsertProfile` is a true upsert: adds the profile if the name is new, replaces in-place if the name already exists. Only error: empty name. |
| Message naming convention | Profiles screen exports its own message types (`profiles.BackMsg`, `profiles.SwitchMsg`, `profiles.ReloadActiveMsg`, `profiles.ConfigChangedMsg`). Root defines the unexported trigger `triggerProfilesMsg`. |
| Smoke tests on Windows | Smoke tests are valid on both Linux and Windows with OS-appropriate binary and path syntax. |

---

## Goal

Implement profile management in the TUI client so the user can create additional
connection profiles, edit any profile, delete profiles, and
switch to a different profile without restarting the application.

The feature must preserve the core EKVS guarantees from `costitution/mission.md`:
- TUI remains the interactive client for managing encrypted key-value stores.
- Authentication continues to rely on SSH key pairs only.
- Switching profile must never carry authentication material across identities.

---

## Scope

### In scope
- Runtime profile CRUD in the TUI.
- A profile-management Bubble Tea screen reachable from the main menu.
- Persistence of profile create/edit/delete operations to the YAML config file.
- Auto-discovery of candidate SSH keys from the platform SSH directory during profile creation.
- Root-model integration for profile switching.
- Clearing all session authentication/encryption state on profile switch.
- Redirect behaviour after deleting the active profile.
- Unit tests for all new or modified TUI packages.

### Out of scope
- SSH agent support (planned for `ssh_agent_support`).
- New theme implementations beyond the existing `adaptive` and `hacker` themes.
- Server-side profile concepts; profiles remain a client-side YAML concern only.
- Remote validation of the selected SSH key or server-side key registration.
- CLI profile management (separate milestone).
- UX polish such as modal overlays, loading spinners, or shortcut help redesign (planned for `tui_ux_polish`).

---

## Home directory resolution

Two helpers are added to `internal/tui/config`:

```go
// ExpandHome replaces a leading "~" with the current user's home directory.
// On Windows the home directory is obtained from os.UserHomeDir().
// The original path is returned unchanged if it does not start with "~".
func ExpandHome(path string) string

// SSHDir returns the path to the user's .ssh directory for the current
// platform (e.g. /home/alice/.ssh on Linux, C:\Users\alice\.ssh on Windows).
// Returns an empty string and an error if the home directory cannot be
// determined.
func SSHDir() (string, error)
```

`ExpandHome` must be called:
- in `DefaultProfile()` so the default `IdentityFile` is already expanded;
- in `applyDefaults` so patched defaults are always expanded;
- in the profiles screen before scanning for candidate SSH keys;
- in the auth layer when loading a private key by path.

`SSHDir` must be used by the profiles screen instead of the hardcoded string `"~/.ssh/"`.

---

## Profile model and persistence

The existing YAML structure introduced in `tui_setup` remains the baseline:

```yaml
profiles:
  - name: "local"
    server_url: "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme: "adaptive"
```

### Persistence rules
- `profiles` remains the only required top-level field.
- No `active_profile` field is introduced.
- The file path used for persistence is the one selected at startup via `--config`
  (default: `ekvs-tui.yaml`).
- If the app started without an existing config file, the first successful
  profile mutation creates it at that same path.
- `config.Save(path, cf)` remains the single write path to disk.
- If more than one profile exists in the YAML file, startup always shows the
  profile-selection screen instead of remembering the last runtime choice.

### Validation rules
- `Profile.Name` must be non-empty.
- `Profile.Name` must be unique within the file.
- `ServerURL`, `IdentityFile`, and `Theme` keep the current defaulting behaviour
  from `internal/tui/config.DefaultProfile()` when omitted or left blank.
- `Theme` values remain constrained to the currently supported set:
  `adaptive`, `hacker`.

---

## Runtime navigation

### Main menu
The main menu exposes `Profiles` as the entry for connection-profile
management, replacing the old `Settings` placeholder.

### Profiles screen
A new screen is introduced for managing profiles.

List mode shows, for every profile:
- name;
- server URL;
- SSH identity file;
- theme;
- active marker (for the profile currently loaded in the root model).

### Keyboard map
- `↑` / `k`: move selection up.
- `↓` / `j`: move selection down.
- `Enter` or `s`: switch to the selected profile.
- `c`: create a new profile.
- `e`: edit the selected profile.
- `d`: delete the selected profile.
- `esc`: return to the main menu.
- `q` / `Ctrl+C`: quit the TUI.

---

## Create profile flow

The create flow collects the same profile fields used by the YAML file:
- `name`
- `server_url`
- `identity_file`
- `theme`

### Requirements
- `name` is mandatory.
- `name` must be unique.
- when collecting `identity_file`, the TUI uses `config.SSHDir()` to obtain the platform SSH directory and proposes the discovered keys as selectable candidates;
- the user must always be able to bypass the discovered list and enter a custom path manually;
- if `config.SSHDir()` fails, the directory is inaccessible, or it yields no candidate keys, profile creation must still work via manual path entry;
- auto-discovery is limited to the platform-standard SSH directory returned by `config.SSHDir()` and does not validate whether the discovered key can authenticate successfully;
- blank `server_url`, `identity_file`, or `theme` values fall back to
  `DefaultProfile()` values.
- on successful save, the new profile is added to the in-memory config and the
  list returns with the newly created profile selected.
- the active profile does not change automatically after creation.

### `identity_file` selection UX during create
The create flow must offer two ways to populate `identity_file`:

1. **Choose from discovered keys**
   - scan the platform SSH directory obtained via `config.SSHDir()` for likely private-key files;
   - present the discovered paths as a selectable list;
   - selecting one fills `identity_file` with that path.

2. **Enter a custom path**
   - always available, even when discovered keys exist;
   - allows specifying a path outside the SSH directory.

The discovery is a convenience mechanism only. The selected or manually entered
path is persisted as plain profile configuration; any later authentication
failure remains part of the existing lazy auth flow.

---

## Edit profile flow

The profiles screen allows editing the currently selected profile.

### Confirmed interpretation
- editing is allowed for any selected profile;
- editable fields are:
  - `name`
  - `server_url`
  - `identity_file`
  - `theme`

### Rename rules
- the edited `name` must remain non-empty;
- the edited `name` must remain unique across all profiles;
- rename support must not rely on `name` being a stable identifier during the
  update operation; the implementation should track the original profile being
  edited via old name, index, or equivalent stable context.

### Save semantics
- saving writes the full updated config file back to disk;
- if the edited profile is currently active, after a successful save the TUI
  always shows a reload prompt (see below), regardless of which fields changed;
- if the edited profile is not active, changes are persisted and the runtime
  theme, session, and active profile remain unchanged.

### Active profile reload prompt
Whenever changes to the **active** profile are saved successfully:
- the config change is persisted first;
- the UI shows a confirmation prompt such as `Reload profile now? [y/N]`;
- if the user confirms, the runtime active profile is switched to the
  just-updated profile using the same session-reset guarantees as a normal
  profile switch (theme re-resolved, session cleared, main menu shown);
- if the user declines, the current runtime profile/session remains active
  until the user explicitly reloads or switches profile later.

This rule applies uniformly: a rename, a `server_url` change, an
`identity_file` change, or a `theme` change on the active profile all trigger
the same reload prompt after a successful save. Simplifying the logic in this
way avoids the need to diff individual fields.

When saving changes to a **non-active** profile:
- the config change is persisted;
- no reload prompt is shown;
- the edited profile does not become active;
- the current runtime profile, theme, and session remain unchanged.

---

## Switch profile flow

Switching to a different profile is the central behavioural requirement of this
milestone.

### Required behaviour
When the user switches from profile `A` to profile `B`:
1. `session.Clear()` is called before `B` becomes active.
2. The root model replaces the active profile with `B`.
3. The root model re-resolves the theme from `B.Theme`.
4. Any screen-specific HTTP clients derived from the old profile are discarded.
5. Control returns to the main menu.
6. Authentication is **not** performed immediately.
7. The next authenticated operation reuses the existing lazy auth flow from
   `tui_auth`.

The same runtime behaviour applies when the user confirms the reload prompt after
editing the currently active profile: the accepted reload is treated as a full
switch to the just-updated profile (session cleared, theme re-resolved, main
menu shown).

### Startup rule
This runtime switch is **not** persisted. On the next application start, if the
config still contains more than one profile, the TUI must show the profile
selection screen again and ask the user which profile to use.

### Security requirement
After a switch, the TUI must no longer retain any authentication material from
profile `A`, including:
- loaded private key signer;
- SSH public key;
- fingerprint;
- derived encryption key stored in `session.Session`.

---

## Delete profile flow

### Deleting a non-active profile
- remove it from the config;
- persist the updated file;
- keep the current active profile unchanged;
- keep the user on the profiles screen.

### Deleting the active profile when other profiles remain
- persist the updated file;
- clear the current session;
- redirect to the profile-selection screen;
- the user must explicitly choose the next active profile.

### Deleting the active profile when no profiles remain
- persist the now-empty (or absent) config state;
- redirect to the existing first-run wizard using its standard flow;
- the wizard's optional-save behaviour is preserved: the user decides whether
  to save and where;
- once the wizard completes (with or without saving), control returns to the
  main menu as it does today.

> **Rationale:** the wizard already handles the first-run case correctly.
> Forcing a different mandatory-save mode would require invasive changes to
> the wizard and the benefit is marginal: if the user declines to save, the
> next restart will simply open the wizard again—the same outcome as any cold
> start with no config file.

---

## Root model integration

`internal/tui/root.Model` becomes the coordinator of runtime profile state.

### Required root responsibilities
- keep the loaded `*tuiconfig.ConfigFile` in memory;
- keep the selected config path in memory;
- instantiate the new profile-management sub-model;
- route messages between the main menu, profile screen, profile-selection
  screen, wizard, auth screen, projects screen, and secrets screen;
- handle session clearing and theme refresh on switch;
- expose `Profiles` as the main-menu entry point for connection-profile management;
- handle the active-profile deletion redirects defined above.

### Required constructor change
`root.New(...)` must receive enough information to support later persistence.
The draft expects the root layer to know:
- the loaded config object (possibly `nil` when no file existed), and
- the config file path used by `cmd/tui/main.go`.

---

## Message contract

The profiles screen lives in `internal/tui/profiles`, a separate package from
`internal/tui/root`. Following the same convention used by `projects` and
`secrets`, the profiles package exports its own message types that root
intercepts. Root defines its own unexported trigger.

### Messages exported by `internal/tui/profiles`

| Type | Payload | When emitted |
|------|---------|--------------|
| `profiles.BackMsg` | — | user presses `Esc` on the profiles screen |
| `profiles.SwitchMsg` | `Profile tuiconfig.Profile` | user selects a different profile with `Enter`/`s` |
| `profiles.ReloadActiveMsg` | `Profile tuiconfig.Profile` | user confirms reload after editing the active profile |
| `profiles.ConfigChangedMsg` | `Config *tuiconfig.ConfigFile` | any CRUD operation persisted successfully |

### Message defined in `internal/tui/root`

| Type | Visibility | When emitted |
|------|-----------|--------------|
| `triggerProfilesMsg` | unexported | `Profiles` menu entry selected |

`profiles.ConfigChangedMsg` carries the full updated `*ConfigFile` so root can
replace its in-memory state atomically after every successful mutation.

`profiles.SwitchMsg` and `profiles.ReloadActiveMsg` are distinct types because
root handles them with slightly different context (switch comes from the list,
reload comes after an edit form save), even though both ultimately perform the
same session-clear + theme-resolve + main-menu sequence.

---

## Package layout

```text
internal/
  tui/
    config/
      config.go
      config_test.go
    profiles/
      profiles.go
      profiles_test.go
    root/
      root.go
      menu.go
      messages.go
      root_test.go
```

The exact filenames inside `internal/tui/profiles` may vary, but the feature
must keep the current package separation style already used for `auth`,
`projects`, `secrets`, and `wizard`.

---

## Testing constraints

From `costitution/roadmap.md` and `costitution/techstack.md`:
- unit tests are mandatory for all new/changed packages;
- use the Go standard `testing` package;
- prefer table-driven tests;
- no integration test is required in this milestone;
- Bubble Tea version remains `charm.land/bubbletea/v2`.

---

## Resolved decisions summary

- Confirming the reload prompt after editing the active profile is equivalent to
  a full switch to the just-saved profile (session cleared, theme re-resolved).
- The reload prompt is shown after saving **any** change to the active profile,
  not only when `identity_file` changes. This eliminates field-level diffing.
- Editing a non-active profile never triggers the reload prompt and never
  changes the current runtime profile, theme, or session.
- During profile creation, `identity_file` candidates are auto-discovered using
  `config.SSHDir()` (platform-aware); manual entry is always allowed.
- `~` in any profile path is expanded at runtime via `config.ExpandHome()`.
- `UpsertProfile` is a true upsert: adds the profile if the name is new,
  replaces it in-place if the name already exists. Only error: empty name.
- `UpdateProfile(oldName string, profile Profile) error` is kept for
  rename-safe edits where the original entry must be located by its old name.
- When the last active profile is deleted, the app redirects to the existing
  first-run wizard without any forced-save modification; the user's choice of
  whether and where to save is preserved.
- Active profile identification in the profiles screen is by name (the name at
  the time the screen is opened). Cursor and active marker are refreshed after
  every CRUD operation that updates the in-memory config.
- Smoke tests are valid on both Linux and Windows; commands use
  OS-appropriate binary names and path separators.







