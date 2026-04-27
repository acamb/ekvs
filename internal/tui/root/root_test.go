package root

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/auth"
	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/profiles"
	"ekvs/internal/tui/theme"
)

func newRootWithProfile(t *testing.T, identityFile string) Model {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "test", ServerURL: "http://localhost:8080", IdentityFile: identityFile, Theme: "adaptive"},
		},
	}
	return New(cfg, "", th)
}

// newRootAuthenticated creates a root model that is already authenticated.
func newRootAuthenticated(t *testing.T, keyPath string) Model {
	t.Helper()
	m := newRootWithProfile(t, keyPath)
	pemBytes, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	fp := internalssh.Fingerprint(pub)
	next, _ := m.Update(auth.AuthSuccessMsg{Signer: signer, PublicKey: pub, Fingerprint: fp})
	return next.(Model)
}

func runRootCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// TestRoot_AuthTrigger verifies that dispatching triggerAuthMsg transitions to screenAuth.
func TestRoot_AuthTrigger(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	if m.screen != screenMain {
		t.Fatalf("expected screenMain, got %v", m.screen)
	}

	next, _ := m.Update(triggerAuthMsg{returnTo: screenMain})
	rm := next.(Model)
	if rm.screen != screenAuth {
		t.Errorf("expected screenAuth after triggerAuthMsg, got %v", rm.screen)
	}
}

// TestRoot_AuthSuccess verifies that AuthSuccessMsg populates the session and
// returns to pendingScreen.
func TestRoot_AuthSuccess(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	m.pendingScreen = screenMain

	pemBytes, err := os.ReadFile("../../../internal/ssh/testdata/ed25519")
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	fp := internalssh.Fingerprint(pub)

	next, _ := m.Update(auth.AuthSuccessMsg{
		Signer:      signer,
		PublicKey:   pub,
		Fingerprint: fp,
	})
	rm := next.(Model)
	if rm.screen != screenMain {
		t.Errorf("expected screenMain after AuthSuccessMsg, got %v", rm.screen)
	}
	if rm.session.Fingerprint != fp {
		t.Errorf("expected fingerprint %q, got %q", fp, rm.session.Fingerprint)
	}
}

// TestRoot_AuthCancel verifies that AuthCancelMsg returns to screenMain without
// establishing a session.
func TestRoot_AuthCancel(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	m.screen = screenAuth

	next, _ := m.Update(auth.AuthCancelMsg{})
	rm := next.(Model)
	if rm.screen != screenMain {
		t.Errorf("expected screenMain after AuthCancelMsg, got %v", rm.screen)
	}
	if rm.session.IsAuthenticated() {
		t.Error("session should not be authenticated after cancel")
	}
}

// ── Task 7: switch semantics and session reset ────────────────────────────────

// TestRoot_SwitchMsg_ClearsSession verifies that SwitchMsg clears the current
// session (signer, fingerprint, encryption key) and returns to the main menu
// with the new profile active.
func TestRoot_SwitchMsg_ClearsSession(t *testing.T) {
	const keyPath = "../../../internal/ssh/testdata/ed25519"
	m := newRootAuthenticated(t, keyPath)

	if !m.session.IsAuthenticated() {
		t.Fatal("precondition: session should be authenticated")
	}
	if m.session.Fingerprint == "" {
		t.Fatal("precondition: fingerprint should be set")
	}

	newProfile := tuiconfig.Profile{Name: "other", ServerURL: "http://other", Theme: "adaptive"}
	next, _ := m.Update(profiles.SwitchMsg{Profile: newProfile})
	rm := next.(Model)

	if rm.session.IsAuthenticated() {
		t.Error("session should be cleared after SwitchMsg")
	}
	if rm.session.Fingerprint != "" {
		t.Errorf("fingerprint should be empty after switch, got %q", rm.session.Fingerprint)
	}
	if rm.session.Signer != nil {
		t.Error("signer should be nil after switch")
	}
	if rm.screen != screenMain {
		t.Errorf("expected screenMain after SwitchMsg, got %v", rm.screen)
	}
	if rm.profile.Name != "other" {
		t.Errorf("expected active profile 'other', got %q", rm.profile.Name)
	}
}

// TestRoot_SwitchMsg_NoImmediateAuth verifies that after a switch the root does
// not immediately trigger the auth flow; the user must perform an authenticated
// action first.
func TestRoot_SwitchMsg_NoImmediateAuth(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")

	newProfile := tuiconfig.Profile{Name: "other", ServerURL: "http://other", Theme: "hacker"}
	next, cmd := m.Update(profiles.SwitchMsg{Profile: newProfile})
	rm := next.(Model)

	if rm.screen == screenAuth {
		t.Error("switch should NOT trigger authentication immediately")
	}
	_ = cmd
}

// TestRoot_SwitchMsg_ReplacesTheme verifies that the theme is re-resolved from
// the new profile after a switch.
func TestRoot_SwitchMsg_ReplacesTheme(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")

	newProfile := tuiconfig.Profile{Name: "other", ServerURL: "http://other", Theme: "hacker"}
	next, _ := m.Update(profiles.SwitchMsg{Profile: newProfile})
	rm := next.(Model)

	if rm.profile.Theme != "hacker" {
		t.Errorf("expected theme 'hacker' after switch, got %q", rm.profile.Theme)
	}
}

// TestRoot_ProfileSwitchMsg_LegacyPath verifies the internal profileSwitchMsg
// (used by the profile-select screen) still clears the session.
func TestRoot_ProfileSwitchMsg_LegacyPath(t *testing.T) {
	const keyPath = "../../../internal/ssh/testdata/ed25519"
	m := newRootAuthenticated(t, keyPath)

	other := tuiconfig.Profile{Name: "other", Theme: "adaptive"}
	next, _ := m.Update(profileSwitchMsg{profile: other})
	rm := next.(Model)

	if rm.session.IsAuthenticated() {
		t.Error("session should be cleared after profileSwitchMsg")
	}
	if rm.screen != screenMain {
		t.Errorf("expected screenMain, got %v", rm.screen)
	}
}

// ── Task 8: handle edits to the active profile ────────────────────────────────

// TestRoot_ReloadActiveMsg_ClearsSessionAndUpdatesConfig verifies that
// ReloadActiveMsg clears the session, updates the active profile, and updates
// the in-memory config.
func TestRoot_ReloadActiveMsg_ClearsSessionAndUpdatesConfig(t *testing.T) {
	const keyPath = "../../../internal/ssh/testdata/ed25519"
	m := newRootAuthenticated(t, keyPath)

	if !m.session.IsAuthenticated() {
		t.Fatal("precondition: session should be authenticated")
	}

	updatedCfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "test-renamed", ServerURL: "http://new", Theme: "hacker"},
		},
	}
	updatedProfile := updatedCfg.Profiles[0]

	next, _ := m.Update(profiles.ReloadActiveMsg{Profile: updatedProfile, Config: updatedCfg})
	rm := next.(Model)

	if rm.session.IsAuthenticated() {
		t.Error("session should be cleared after ReloadActiveMsg")
	}
	if rm.session.Fingerprint != "" {
		t.Errorf("fingerprint should be empty after reload, got %q", rm.session.Fingerprint)
	}
	if rm.profile.Name != "test-renamed" {
		t.Errorf("active profile not updated; want 'test-renamed', got %q", rm.profile.Name)
	}
	if rm.config != updatedCfg {
		t.Error("root config not updated after ReloadActiveMsg")
	}
	if rm.screen != screenMain {
		t.Errorf("expected screenMain after ReloadActiveMsg, got %v", rm.screen)
	}
}

// TestRoot_ConfigChangedMsg_UpdatesConfig verifies that ConfigChangedMsg
// updates the root in-memory config and stays on the profiles screen.
func TestRoot_ConfigChangedMsg_UpdatesConfig(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "a", Theme: "adaptive"},
			{Name: "b", Theme: "hacker"},
		},
	}
	m := New(cfg, "", th)
	// Put root on the profiles screen.
	m.screen = screenProfiles
	m.profilesModel = profiles.New(cfg, "", "a", th)

	newCfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "a", Theme: "adaptive"},
			{Name: "b-renamed", Theme: "hacker"},
		},
	}
	next, _ := m.Update(profiles.ConfigChangedMsg{Config: newCfg})
	rm := next.(Model)

	if rm.config != newCfg {
		t.Error("root config not updated after ConfigChangedMsg")
	}
	if rm.screen != screenProfiles {
		t.Errorf("expected screenProfiles after non-delete ConfigChangedMsg, got %v", rm.screen)
	}
}

// TestRoot_ConfigChangedMsg_ActiveDeleted_LastProfile redirects to wizard.
func TestRoot_ConfigChangedMsg_ActiveDeleted_LastProfile(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{Profiles: []tuiconfig.Profile{{Name: "only", Theme: "adaptive"}}}
	m := New(cfg, "", th)

	emptyCfg := &tuiconfig.ConfigFile{}
	next, _ := m.Update(profiles.ConfigChangedMsg{Config: emptyCfg, ActiveDeleted: true})
	rm := next.(Model)

	if rm.screen != screenWizard {
		t.Errorf("expected screenWizard when last profile deleted, got %v", rm.screen)
	}
}

// TestRoot_ConfigChangedMsg_ActiveDeleted_OthersRemain redirects to profile selection.
func TestRoot_ConfigChangedMsg_ActiveDeleted_OthersRemain(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "a", Theme: "adaptive"},
			{Name: "b", Theme: "hacker"},
		},
	}
	m := New(cfg, "", th)

	remaining := &tuiconfig.ConfigFile{Profiles: []tuiconfig.Profile{{Name: "b", Theme: "hacker"}}}
	next, _ := m.Update(profiles.ConfigChangedMsg{Config: remaining, ActiveDeleted: true})
	rm := next.(Model)

	if rm.screen != screenProfileSelect {
		t.Errorf("expected screenProfileSelect when active deleted with remaining profiles, got %v", rm.screen)
	}
}

// TestRoot_Profiles_MenuEntry verifies that the main menu contains "Profiles"
// and does not contain "Settings".
func TestRoot_Profiles_MenuEntry(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	found := false
	for _, item := range m.main.items {
		if item.ID == "profiles" {
			found = true
		}
		if item.ID == "settings" {
			t.Error("main menu should not contain 'Settings' item")
		}
	}
	if !found {
		t.Error("main menu should contain 'Profiles' item")
	}
}

// TestRoot_TriggerProfilesMsg_TransitionsToProfilesScreen verifies that
// selecting Profiles from the main menu transitions to screenProfiles.
func TestRoot_TriggerProfilesMsg_TransitionsToProfilesScreen(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")

	next, _ := m.Update(triggerProfilesMsg{})
	rm := next.(Model)

	if rm.screen != screenProfiles {
		t.Errorf("expected screenProfiles after triggerProfilesMsg, got %v", rm.screen)
	}
}

// TestRoot_ProfilesBack_ReturnsToMain verifies that BackMsg from the profiles
// screen returns to the main menu.
func TestRoot_ProfilesBack_ReturnsToMain(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	m.screen = screenProfiles

	next, _ := m.Update(profiles.BackMsg{})
	rm := next.(Model)

	if rm.screen != screenMain {
		t.Errorf("expected screenMain after profiles.BackMsg, got %v", rm.screen)
	}
}

// TestRoot_MultipleProfiles_ShowsProfileSelection verifies that starting the
// application with more than one configured profile shows the profile-selection
// screen, simulating a restart with an existing multi-profile config file.
func TestRoot_MultipleProfiles_ShowsProfileSelection(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "local", ServerURL: "http://localhost:8080", Theme: "adaptive"},
			{Name: "prod", ServerURL: "https://prod.example.com", Theme: "hacker"},
		},
	}
	m := New(cfg, "", th)

	if m.screen != screenProfileSelect {
		t.Errorf("expected screenProfileSelect on startup with multiple profiles, got %v", m.screen)
	}
}

// TestRoot_SingleProfile_SkipsProfileSelection verifies that starting with a
// single profile skips the selection screen and goes directly to the main menu.
func TestRoot_SingleProfile_SkipsProfileSelection(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "only", ServerURL: "http://localhost:8080", Theme: "adaptive"},
		},
	}
	m := New(cfg, "", th)

	if m.screen != screenMain {
		t.Errorf("expected screenMain on startup with single profile, got %v", m.screen)
	}
}

// TestRoot_NoProfiles_ShowsWizard verifies that starting with no profiles
// (nil config) shows the first-run wizard.
func TestRoot_NoProfiles_ShowsWizard(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	m := New(nil, "", th)

	if m.screen != screenWizard {
		t.Errorf("expected screenWizard on startup with no config, got %v", m.screen)
	}
}

// TestRoot_WindowSizeMsg_UpdatesDimensions verifies that a tea.WindowSizeMsg
// updates the width and height fields of the root model.
func TestRoot_WindowSizeMsg_UpdatesDimensions(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")

	if m.width != 0 || m.height != 0 {
		t.Fatalf("expected zero dimensions before first WindowSizeMsg, got %dx%d", m.width, m.height)
	}

	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rm := next.(Model)

	if rm.width != 120 {
		t.Errorf("expected width=120, got %d", rm.width)
	}
	if rm.height != 40 {
		t.Errorf("expected height=40, got %d", rm.height)
	}
}

// TestRoot_WindowSizeMsg_Propagated verifies that a tea.WindowSizeMsg is
// forwarded to the active sub-model (here: profilesModel which stores width).
func TestRoot_WindowSizeMsg_Propagated(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "a", Theme: "adaptive"},
		},
	}
	m := New(cfg, "", th)
	m.screen = screenProfiles
	m.profilesModel = profiles.New(cfg, "", "a", th)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	rm := next.(Model)

	// The root must have updated its own dimensions.
	if rm.width != 100 {
		t.Errorf("root width not updated: want 100, got %d", rm.width)
	}
	// The profiles model receives the message through the dispatch loop.
	// We verify it did not panic and the root dimensions are correct.
	// (profiles.Model will expose width after Task 8; here we just smoke-test propagation.)
	if rm.screen != screenProfiles {
		t.Errorf("screen changed unexpectedly: got %v", rm.screen)
	}
}

// TestRoot_TriggerProfilesMsg_InjectsWidth verifies that when the root model
// transitions to screenProfiles it immediately injects the already-known
// terminal width into profilesModel, so the split layout is visible on first
// render without requiring a resize event.
func TestRoot_TriggerProfilesMsg_InjectsWidth(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "a", Theme: "adaptive"},
		},
	}
	m := New(cfg, "", th)
	// Simulate a WindowSizeMsg that arrived before the user opened profiles.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(Model)

	// Now trigger the profiles screen.
	next, _ = m.Update(triggerProfilesMsg{})
	rm := next.(Model)

	if rm.screen != screenProfiles {
		t.Fatalf("want screenProfiles, got %v", rm.screen)
	}
	// The profiles model must already know the terminal width.
	if rm.profilesModel.Width() != 120 {
		t.Errorf("profilesModel.Width(): want 120, got %d", rm.profilesModel.Width())
	}
}

// TestRoot_ConfigChangedMsg_InjectsWidth verifies that re-creating profilesModel
// on ConfigChangedMsg also injects the current root width.
func TestRoot_ConfigChangedMsg_InjectsWidth(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "a", Theme: "adaptive"},
		},
	}
	m := New(cfg, "", th)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(Model)
	m.screen = screenProfiles

	// Simulate a non-active profile update.
	next, _ = m.Update(profiles.ConfigChangedMsg{Config: cfg})
	rm := next.(Model)

	if rm.profilesModel.Width() != 80 {
		t.Errorf("profilesModel.Width() after ConfigChangedMsg: want 80, got %d", rm.profilesModel.Width())
	}
}
