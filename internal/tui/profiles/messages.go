// Package profiles implements the Bubble Tea model for the EKVS profile-management screen.
package profiles

import tuiconfig "ekvs/internal/tui/config"

// BackMsg is emitted when the user exits the Profiles screen (Esc from list mode).
// The root model handles it by returning to the main menu without any state change.
type BackMsg struct{}

// SwitchMsg is emitted when the user selects a profile that is not currently
// active. The root model must:
//  1. call session.Clear();
//  2. replace the active profile with Profile;
//  3. re-resolve the theme;
//  4. discard any screen-specific HTTP clients;
//  5. navigate to the main menu (no immediate authentication).
type SwitchMsg struct {
	Profile tuiconfig.Profile
}

// ReloadActiveMsg is emitted when the user has edited the currently active
// profile and confirmed the reload prompt. The root model handles this
// identically to SwitchMsg: session cleared, theme re-resolved, and the main
// menu is shown with the updated profile active.
type ReloadActiveMsg struct {
	Profile tuiconfig.Profile
}

// ConfigChangedMsg is emitted after any successful CRUD operation (create,
// edit, or delete). The root model must update its in-memory *ConfigFile
// reference to Config.
//
// ActiveDeleted is true when the profile that was just deleted was the
// currently active one; root uses this flag to decide where to navigate next:
//   - false → stay on the profiles screen (cursor already adjusted);
//   - true, remaining profiles exist → navigate to the profile-selection screen;
//   - true, no profiles remain → navigate to the first-run wizard.
type ConfigChangedMsg struct {
	Config        *tuiconfig.ConfigFile
	ActiveDeleted bool
}
