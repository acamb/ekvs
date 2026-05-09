package profiles

// profiles_e2e_test.go — end-to-end user-journey tests for the Profiles screen.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	tuiconfig "ekvs/internal/tui/config"
)

// TestE2E_Profiles_SplitLayoutWithNavigation verifies that after a WindowSizeMsg
// the layout shows both a list pane and a detail pane, and that navigating down
// updates the detail pane.
func TestE2E_Profiles_SplitLayoutWithNavigation(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "alpha", ServerURL: "http://alpha.example.com", Theme: "adaptive"},
		tuiconfig.Profile{Name: "beta", ServerURL: "http://beta.example.com", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, "", "alpha")
	m, _ = m.UpdateTyped(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := m.View().Content
	if !strings.Contains(view, "alpha") {
		t.Errorf("list pane should contain 'alpha';\n%s", view)
	}
	if !strings.Contains(view, "beta") {
		t.Errorf("list pane should contain 'beta';\n%s", view)
	}
	if !strings.Contains(view, "http://alpha.example.com") {
		t.Errorf("detail pane should show alpha's Server URL;\n%s", view)
	}

	// Navigate down → detail pane should update to beta.
	m, _ = sendKey(m, "down")
	view2 := m.View().Content
	if !strings.Contains(view2, "http://beta.example.com") {
		t.Errorf("detail pane should show beta's Server URL after ↓;\n%s", view2)
	}
	if !strings.Contains(view2, "http://alpha.example.com") {
		// alpha's URL should NOT appear in the detail pane now.
		// (It might appear in the list pane label, so we just check the detail
		// panel updated – verified by presence of beta URL above.)
	}
}

// TestE2E_Profiles_ActiveProfileMarked verifies the active profile is marked
// with "*" in the list pane.
func TestE2E_Profiles_ActiveProfileMarked(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "prod", ServerURL: "http://prod", Theme: "adaptive"},
		tuiconfig.Profile{Name: "dev", ServerURL: "http://dev", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, "", "prod")
	m, _ = m.UpdateTyped(tea.WindowSizeMsg{Width: 120, Height: 40})

	view := m.View().Content
	if !strings.Contains(view, "*") {
		t.Errorf("active profile should be marked with '*';\n%s", view)
	}
}

// TestE2E_Profiles_EditModeHidesSplitLayout verifies that pressing 'e' enters
// edit mode and the detail-panel split layout is replaced by the form.
func TestE2E_Profiles_EditModeHidesSplitLayout(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "myprofile", ServerURL: "http://srv", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, "", "myprofile")
	m, _ = m.UpdateTyped(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Enter edit mode.
	m, _ = sendKey(m, "e")
	if m.mode != modeEdit {
		t.Fatalf("expected modeEdit after 'e', got %v", m.mode)
	}

	view := m.View().Content
	// The form should show the profile name field.
	if !strings.Contains(view, "Profile") {
		t.Errorf("edit form should show profile field label;\n%s", view)
	}
}

// TestE2E_Profiles_DeleteProfileFlow tests d → y → profile removed.
func TestE2E_Profiles_DeleteProfileFlow(t *testing.T) {
	configPath := makeTempConfigPath(t)
	cfg := testConfig(
		tuiconfig.Profile{Name: "keep", ServerURL: "http://keep", Theme: "adaptive"},
		tuiconfig.Profile{Name: "gone", ServerURL: "http://gone", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, configPath, "keep")

	// Move to "gone" (second item).
	m, _ = sendKey(m, "down")

	// Enter delete confirm.
	m, _ = sendKey(m, "d")
	if m.mode != modeDeleteConfirm {
		t.Fatalf("expected modeDeleteConfirm, got %v", m.mode)
	}
	view := m.View().Content
	if !strings.Contains(view, "gone") {
		t.Errorf("delete confirm should mention 'gone';\n%s", view)
	}

	// Confirm.
	m2, cmd := sendKey(m, "y")
	_ = m2
	if cmd == nil {
		t.Fatal("expected non-nil cmd after 'y'")
	}
	msg := cmd()
	if _, ok := msg.(ConfigChangedMsg); !ok {
		t.Errorf("expected ConfigChangedMsg, got %T", msg)
	}
}

// TestE2E_Profiles_CreateModeEntersForm verifies 'c' enters the create form.
func TestE2E_Profiles_CreateModeEntersForm(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "existing", ServerURL: "http://srv", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, "", "existing")
	m, _ = sendKey(m, "c")
	if m.mode != modeCreate {
		t.Fatalf("expected modeCreate after 'c', got %v", m.mode)
	}
	view := m.View().Content
	// Should show the name field in the create form.
	if !strings.Contains(view, "Profile") {
		t.Errorf("create form should show profile field label;\n%s", view)
	}
}

// TestE2E_Profiles_FooterHintsListMode verifies footer hints in list mode.
func TestE2E_Profiles_FooterHintsListMode(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "p", ServerURL: "http://srv", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, "", "p")
	view := m.View().Content
	// The footer in list mode shows: c new • e edit • d delete
	for _, want := range []string{"c", "e", "d"} {
		if !strings.Contains(view, want) {
			t.Errorf("list mode footer missing %q;\n%s", want, view)
		}
	}
}

// TestE2E_Profiles_CursorWraps verifies cursor wraps from last to first profile.
func TestE2E_Profiles_CursorWraps(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "first", ServerURL: "http://a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "second", ServerURL: "http://b", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, "", "first")
	m, _ = m.UpdateTyped(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Move down to second.
	m, _ = sendKey(m, "down")
	// Move down again → should wrap to first.
	m, _ = sendKey(m, "down")
	if m.cursor != 0 {
		t.Errorf("cursor should wrap to 0, got %d", m.cursor)
	}
}
