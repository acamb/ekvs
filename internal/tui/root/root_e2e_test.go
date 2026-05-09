package root

// root_e2e_test.go — end-to-end user-journey tests for the root model.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/theme"
)

const e2eKeyPath = "../../../internal/ssh/testdata/ed25519"

// TestE2E_Root_MainMenuContents verifies the main menu view contains expected items.
func TestE2E_Root_MainMenuContents(t *testing.T) {
	m := newRootWithProfile(t, e2eKeyPath)
	view := m.View().Content
	for _, want := range []string{"Projects", "Profiles", "Quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("main menu should contain %q;\n%s", want, view)
		}
	}
}

// TestE2E_Root_MenuNavigationChangesHighlight verifies that cursor moves
// through menu items.
func TestE2E_Root_MenuNavigationChangesHighlight(t *testing.T) {
	m := newRootWithProfile(t, e2eKeyPath)

	// Initially cursor=0: Projects should be the only item with ">".
	if m.screen != screenMain {
		t.Fatalf("expected screenMain, got %v", m.screen)
	}
	if m.main.cursor != 0 {
		t.Fatalf("initial cursor should be 0, got %d", m.main.cursor)
	}

	// Press ↓ → cursor moves to Profiles.
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = next.(Model)
	if m.main.cursor != 1 {
		t.Errorf("cursor should be 1 after ↓, got %d", m.main.cursor)
	}

	// Press ↑ → back to Projects.
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = next.(Model)
	if m.main.cursor != 0 {
		t.Errorf("cursor should be 0 after ↑, got %d", m.main.cursor)
	}
}

// TestE2E_Root_ProfileSelectShowsMultipleProfiles verifies that with two
// profiles the profile selection screen shows both names.
func TestE2E_Root_ProfileSelectShowsMultipleProfiles(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "production", ServerURL: "http://prod", IdentityFile: e2eKeyPath, Theme: "adaptive"},
			{Name: "staging", ServerURL: "http://stag", IdentityFile: e2eKeyPath, Theme: "adaptive"},
		},
	}
	m := New(cfg, "", th)
	if m.screen != screenProfileSelect {
		t.Fatalf("expected screenProfileSelect with 2 profiles, got %v", m.screen)
	}
	view := m.View().Content
	for _, want := range []string{"production", "staging"} {
		if !strings.Contains(view, want) {
			t.Errorf("profile select should contain %q;\n%s", want, view)
		}
	}
}

// TestE2E_Root_WindowResizeUpdatesModelAndView verifies WindowSizeMsg updates
// width/height and view remains non-empty.
func TestE2E_Root_WindowResizeUpdatesModelAndView(t *testing.T) {
	m := newRootWithProfile(t, e2eKeyPath)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	m = next.(Model)
	if m.width != 200 {
		t.Errorf("width = %d, want 200", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
	if len(m.View().Content) == 0 {
		t.Error("view should not be empty after resize")
	}
}

// TestE2E_Root_FooterVisibleInMainMenu verifies the main menu footer bar.
func TestE2E_Root_FooterVisibleInMainMenu(t *testing.T) {
	m := newRootWithProfile(t, e2eKeyPath)
	view := m.View().Content
	if !strings.Contains(view, "navigate") || !strings.Contains(view, "quit") {
		t.Errorf("main menu should show navigation footer hints;\n%s", view)
	}
}

// TestE2E_Root_WizardShownWhenNoProfiles verifies the wizard is active when
// there is no config.
func TestE2E_Root_WizardShownWhenNoProfiles(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	m := New(nil, "", th)
	if m.screen != screenWizard {
		t.Fatalf("expected screenWizard with no config, got %v", m.screen)
	}
	view := m.View().Content
	// The wizard first step shows "Profile name:" label.
	if !strings.Contains(view, "Profile") {
		t.Errorf("wizard view should show profile prompt;\n%s", view)
	}
}

// TestE2E_Root_SingleProfileSkipsSelect verifies that one profile goes straight
// to the main menu (no profile selection screen).
func TestE2E_Root_SingleProfileSkipsSelect(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "only", ServerURL: "http://srv", IdentityFile: e2eKeyPath, Theme: "adaptive"},
		},
	}
	m := New(cfg, "", th)
	if m.screen != screenMain {
		t.Fatalf("expected screenMain with 1 profile, got %v", m.screen)
	}
	view := m.View().Content
	if !strings.Contains(view, "Projects") {
		t.Errorf("main menu should be shown;\n%s", view)
	}
}

// TestE2E_Root_AuthenticatedSessionNavigatesToProjects verifies that an already
// authenticated root model can navigate to the projects screen via triggerProjectsMsg.
func TestE2E_Root_AuthenticatedSessionNavigatesToProjects(t *testing.T) {
	m := newRootAuthenticated(t, e2eKeyPath)
	if !m.session.IsAuthenticated() {
		t.Fatal("session should be authenticated")
	}

	// Trigger the projects screen directly.
	next, cmd := m.Update(triggerProjectsMsg{})
	m = next.(Model)
	if m.screen != screenProjects {
		t.Errorf("expected screenProjects after triggerProjectsMsg, got %v", m.screen)
	}
	// Init the projects model so its fetch runs.
	if cmd != nil {
		// The init cmd from projectsModel.Init() — don't run it to avoid HTTP call.
	}
	view := m.View().Content
	if !strings.Contains(view, "Projects") {
		t.Errorf("projects screen should show 'Projects' title;\n%s", view)
	}
}
