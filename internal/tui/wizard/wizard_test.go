package wizard

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/theme"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func testTheme(t *testing.T) theme.Theme {
	t.Helper()
	th, err := theme.NewTheme("adaptive")
	if err != nil {
		t.Fatalf("theme.NewTheme: %v", err)
	}
	return th
}

// fakeSSHDir returns a sshDirFn that points to a temp directory pre-populated
// with the given file names.
func fakeSSHDir(t *testing.T, names ...string) func() (string, error) {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("dummy"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return func() (string, error) { return dir, nil }
}

// press sends a single key to the model and returns the updated model.
func press(m Model, key string) Model {
	updated, _ := m.UpdateTyped(keyMsg(key))
	return updated
}

// keyMsg builds a tea.KeyPressMsg whose String() matches key.
func keyMsg(key string) tea.Msg {
	switch key {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	default:
		if len(key) == 1 {
			r := []rune(key)[0]
			return tea.KeyPressMsg{Code: r, Text: key}
		}
		return tea.KeyPressMsg{}
	}
}

// advanceToIdentityStep types a profile name and presses Enter twice to reach
// stepIdentityFile.
func advanceToIdentityStep(t *testing.T, m Model) Model {
	t.Helper()
	for _, ch := range "myprofile" {
		m = press(m, string(ch))
	}
	m = press(m, "enter") // stepName → stepServerURL
	m = press(m, "enter") // stepServerURL → stepIdentityFile
	if m.step != stepIdentityFile {
		t.Fatalf("expected stepIdentityFile, got step %d", m.step)
	}
	return m
}

// ── selectedIdentity ──────────────────────────────────────────────────────────

func TestSelectedIdentity_PickMode(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "id_ed25519", "id_rsa"))
	// Default: pick mode, cursor at 0 → first key selected.
	got := m.selectedIdentity()
	if filepath.Base(got) != "id_ed25519" {
		t.Errorf("want id_ed25519, got %q", got)
	}
}

func TestSelectedIdentity_ManualMode(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t)) // no keys → manual
	// Manual mode with empty value.
	if got := m.selectedIdentity(); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}

func TestSelectedIdentity_PickCursor(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "id_ed25519", "id_rsa"))
	m = advanceToIdentityStep(t, m)
	m = press(m, "down") // cursor → 1
	got := m.selectedIdentity()
	if filepath.Base(got) != "id_rsa" {
		t.Errorf("want id_rsa, got %q", got)
	}
}

// ── no keys → manual ─────────────────────────────────────────────────────────

func TestNoKeysDefaultsToManualMode(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t)) // empty dir
	if m.identMode != identityModeManual {
		t.Error("expected manual mode when no keys found")
	}
}

// ── pick mode cursor navigation ───────────────────────────────────────────────

func TestPickMode_DownWraps(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "a", "b", "c"))
	m = advanceToIdentityStep(t, m)

	m = press(m, "down") // 0→1
	m = press(m, "down") // 1→2
	m = press(m, "down") // 2→0 (wrap)
	if m.discoveryCursor != 0 {
		t.Errorf("wrap-around down: want 0, got %d", m.discoveryCursor)
	}
}

func TestPickMode_UpWraps(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "a", "b", "c"))
	m = advanceToIdentityStep(t, m)

	m = press(m, "up") // 0→2 (wrap)
	if m.discoveryCursor != 2 {
		t.Errorf("wrap-around up: want 2, got %d", m.discoveryCursor)
	}
}

func TestPickMode_jkNavigation(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "a", "b"))
	m = advanceToIdentityStep(t, m)

	m = press(m, "j") // 0→1
	if m.discoveryCursor != 1 {
		t.Errorf("j: want 1, got %d", m.discoveryCursor)
	}
	m = press(m, "k") // 1→0
	if m.discoveryCursor != 0 {
		t.Errorf("k: want 0, got %d", m.discoveryCursor)
	}
}

// ── m key toggles to manual ───────────────────────────────────────────────────

func TestMKey_SwitchesToManual(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "id_ed25519"))
	m = advanceToIdentityStep(t, m)
	if m.identMode != identityModePick {
		t.Fatal("expected pick mode before m")
	}
	m = press(m, "m")
	if m.identMode != identityModeManual {
		t.Error("expected manual mode after m")
	}
}

// ── esc in manual → returns to pick (keys exist) ─────────────────────────────

func TestEsc_ManualToPick_WhenKeysExist(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "id_ed25519"))
	m = advanceToIdentityStep(t, m)
	m = press(m, "m")   // → manual
	m = press(m, "esc") // → pick (because keys exist)
	if m.identMode != identityModePick {
		t.Error("esc in manual should return to pick when keys exist")
	}
	if m.step != stepIdentityFile {
		t.Errorf("step should remain stepIdentityFile, got %d", m.step)
	}
}

// ── esc in manual (no keys) → goes to previous step ──────────────────────────

func TestEsc_ManualNoPick_GoesToPreviousStep(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t)) // no keys → manual
	m = advanceToIdentityStep(t, m)
	m = press(m, "esc")
	if m.step != stepServerURL {
		t.Errorf("expected stepServerURL, got step %d", m.step)
	}
}

// ── enter confirms and advances to stepConfirmSave ────────────────────────────

func TestEnter_AdvancesFromIdentityTpConfirmSave(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "id_ed25519"))
	m = advanceToIdentityStep(t, m)
	m = press(m, "enter")
	if m.step != stepConfirmSave {
		t.Errorf("expected stepConfirmSave, got step %d", m.step)
	}
}

// ── selectedIdentity used in buildProfile (via finish) ───────────────────────

func TestFinish_UsesSelectedIdentity(t *testing.T) {
	m := NewModel(testTheme(t)).WithSSHDirFn(fakeSSHDir(t, "id_ed25519"))
	m = advanceToIdentityStep(t, m)
	m = press(m, "down")  // select id_ed25519 (only key, stays at 0 — or move if 2+ keys)
	m = press(m, "enter") // → stepConfirmSave
	// Press N (no save) to trigger finish without file I/O.
	m, cmd := m.UpdateTyped(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd == nil {
		t.Fatal("expected a command from finish()")
	}
	// The pendingProfile should use the discovered key.
	if filepath.Base(m.pendingProfile.IdentityFile) != "id_ed25519" {
		t.Errorf("pendingProfile.IdentityFile: want id_ed25519, got %q", m.pendingProfile.IdentityFile)
	}
}
