package wizard

// wizard_e2e_test.go — end-to-end user-journey tests for the Wizard screen.

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_Wizard_InitialViewShowsNamePrompt verifies the view on first rendering.
func TestE2E_Wizard_InitialViewShowsNamePrompt(t *testing.T) {
	m := NewModel(testTheme(t))
	view := m.View().Content
	// The wizard initial step shows "Profile name:" label.
	if !strings.Contains(view, "Profile") {
		t.Errorf("initial view should show profile prompt;\n%s", view)
	}
}

// TestE2E_Wizard_FullCompletionFlowNoSave drives the wizard through all steps
// (name → serverURL → identity → confirmSave=n) and verifies DoneMsg is emitted
// with the entered profile data.
func TestE2E_Wizard_FullCompletionFlowNoSave(t *testing.T) {
	sshDir := fakeSSHDir(t, "id_ed25519")
	m := NewModel(testTheme(t)).WithSSHDirFn(sshDir)

	// Step 1: type profile name.
	for _, ch := range "myprod" {
		m = press(m, string(ch))
	}
	m = press(m, "enter") // → stepServerURL
	if m.step != stepServerURL {
		t.Fatalf("expected stepServerURL, got step %d", m.step)
	}

	// Step 2: accept the default server URL (just press Enter without typing).
	m = press(m, "enter") // → stepIdentityFile
	if m.step != stepIdentityFile {
		t.Fatalf("expected stepIdentityFile, got step %d", m.step)
	}

	// Step 3: in pick mode, press Enter to confirm selected key.
	m = press(m, "enter") // → stepConfirmSave
	if m.step != stepConfirmSave {
		t.Fatalf("expected stepConfirmSave, got step %d", m.step)
	}

	// Confirm save prompt view.
	view := m.View().Content
	if !strings.Contains(strings.ToLower(view), "save") {
		t.Errorf("stepConfirmSave view should ask about saving;\n%s", view)
	}

	// Step 4: press 'n' (no save) → finish() → DoneMsg command.
	m2, cmd := m.UpdateTyped(keyMsg("n"))
	_ = m2
	if cmd == nil {
		t.Fatal("expected non-nil cmd after 'n' at confirmSave")
	}
	msg := cmd()
	done, ok := msg.(DoneMsg)
	if !ok {
		t.Fatalf("expected DoneMsg, got %T", msg)
	}
	if done.Profile.Name != "myprod" {
		t.Errorf("profile name = %q, want 'myprod'", done.Profile.Name)
	}
	if filepath.Base(done.Profile.IdentityFile) != "id_ed25519" {
		t.Errorf("identity file = %q, want 'id_ed25519'", done.Profile.IdentityFile)
	}
}

// TestE2E_Wizard_PickListVisibleWhenKeysFound verifies that with discovered SSH
// keys the identity step shows a pick list.
func TestE2E_Wizard_PickListVisibleWhenKeysFound(t *testing.T) {
	sshDir := fakeSSHDir(t, "id_ed25519", "id_rsa")
	m := NewModel(testTheme(t)).WithSSHDirFn(sshDir)
	m = advanceToIdentityStep(t, m)

	view := m.View().Content
	if !strings.Contains(view, "id_ed25519") {
		t.Errorf("pick list should contain 'id_ed25519';\n%s", view)
	}
	if !strings.Contains(view, "id_rsa") {
		t.Errorf("pick list should contain 'id_rsa';\n%s", view)
	}
}

// TestE2E_Wizard_PickListNavigationUpdatesView verifies cursor movement in pick mode.
func TestE2E_Wizard_PickListNavigationUpdatesView(t *testing.T) {
	sshDir := fakeSSHDir(t, "key_a", "key_b")
	m := NewModel(testTheme(t)).WithSSHDirFn(sshDir)
	m = advanceToIdentityStep(t, m)

	// Initially cursor=0 → key_a selected.
	if filepath.Base(m.selectedIdentity()) != "key_a" {
		t.Errorf("initial selection = %q, want 'key_a'", filepath.Base(m.selectedIdentity()))
	}

	// Press ↓ → key_b selected.
	m = press(m, "down")
	if filepath.Base(m.selectedIdentity()) != "key_b" {
		t.Errorf("after ↓ selection = %q, want 'key_b'", filepath.Base(m.selectedIdentity()))
	}
}

// TestE2E_Wizard_SwitchToManualAndBackFlow tests m → manual view → Esc → pick list.
func TestE2E_Wizard_SwitchToManualAndBackFlow(t *testing.T) {
	sshDir := fakeSSHDir(t, "id_ed25519")
	m := NewModel(testTheme(t)).WithSSHDirFn(sshDir)
	m = advanceToIdentityStep(t, m)

	// Switch to manual mode.
	m = press(m, "m")
	if m.identMode != identityModeManual {
		t.Fatalf("expected identityModeManual, got %v", m.identMode)
	}
	view := m.View().Content
	if !strings.Contains(view, "custom") && !strings.Contains(view, "path") && !strings.Contains(view, "Enter") {
		t.Errorf("manual mode view should show manual input hints;\n%s", view)
	}

	// Esc returns to pick mode.
	m = press(m, "esc")
	if m.identMode != identityModePick {
		t.Fatalf("expected identityModePick after Esc, got %v", m.identMode)
	}
	view2 := m.View().Content
	if !strings.Contains(view2, "id_ed25519") {
		t.Errorf("pick mode should show the discovered key;\n%s", view2)
	}
}

// TestE2E_Wizard_EmptyNameSkipsAdvance verifies that pressing Enter on an empty
// name field does not advance the step.
func TestE2E_Wizard_EmptyNameSkipsAdvance(t *testing.T) {
	m := NewModel(testTheme(t))
	m = press(m, "enter") // empty name
	if m.step != stepName {
		t.Errorf("empty name should keep step at stepName, got step %d", m.step)
	}
}
