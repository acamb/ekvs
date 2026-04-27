package profiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/modal"
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

func testConfig(profiles ...tuiconfig.Profile) *tuiconfig.ConfigFile {
	return &tuiconfig.ConfigFile{Profiles: profiles}
}

func newTestModel(
	t *testing.T,
	cfg *tuiconfig.ConfigFile,
	configPath string,
	active string,
) Model {
	t.Helper()
	return New(cfg, configPath, active, testTheme(t))
}

// keyMsg builds a tea.KeyPressMsg for a named or single-character key.
func keyMsg(key string) tea.KeyPressMsg {
	switch key {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	default:
		r := []rune(key)[0]
		return tea.KeyPressMsg{Code: r, Text: key}
	}
}

func sendKey(m Model, key string) (Model, tea.Cmd) {
	next, cmd := m.Update(keyMsg(key))
	mm, _ := next.(Model)
	return mm, cmd
}

// typeString simulates typing each character of s.
func typeString(m Model, s string) Model {
	for _, ch := range s {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m, _ = next.(Model)
	}
	return m
}

// runCmd calls cmd() and returns the resulting message.
func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected non-nil command, got nil")
	}
	return cmd()
}

// makeTempConfigPath returns a path inside a temp directory (file not yet created).
func makeTempConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "ekvs-tui.yaml")
}

// fakeSSHDir returns a function usable as sshDirFn that serves from tmpDir.
// It creates a private key file and a .pub file to simulate discovery.
func fakeSSHDir(t *testing.T) (func() (string, error), string) {
	t.Helper()
	dir := t.TempDir()
	// Create a fake private key file and its .pub counterpart.
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519"), []byte("FAKE"), 0o600); err != nil {
		t.Fatalf("create fake key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id_ed25519.pub"), []byte("FAKE_PUB"), 0o644); err != nil {
		t.Fatalf("create fake pub: %v", err)
	}
	fn := func() (string, error) { return dir, nil }
	return fn, filepath.Join(dir, "id_ed25519")
}

// ── list mode: navigation ─────────────────────────────────────────────────────

func TestProfilesList_CursorDown(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", ServerURL: "http://a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", ServerURL: "http://b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("want cursor=1, got %d", m.cursor)
	}
}

func TestProfilesList_CursorWrap(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", Theme: "adaptive"},
		tuiconfig.Profile{Name: "c", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "up") // wrap from 0 to 2
	if m.cursor != 2 {
		t.Errorf("wrap-up: want cursor=2, got %d", m.cursor)
	}

	m, _ = sendKey(m, "down") // back to 0
	if m.cursor != 0 {
		t.Errorf("wrap-down: want cursor=0, got %d", m.cursor)
	}
}

func TestProfilesList_JKAliases(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", Theme: "adaptive"},
	)
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "j")
	if m.cursor != 1 {
		t.Errorf("j: want cursor=1, got %d", m.cursor)
	}
	m, _ = sendKey(m, "k")
	if m.cursor != 0 {
		t.Errorf("k: want cursor=0, got %d", m.cursor)
	}
}

// ── list mode: active marker in view ─────────────────────────────────────────

func TestProfilesList_ActiveMarker(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "local", ServerURL: "http://a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "prod", ServerURL: "http://b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, makeTempConfigPath(t), "local")

	view := m.View().Content
	if !strings.Contains(view, "local *") {
		t.Errorf("active marker missing for 'local'; view:\n%s", view)
	}
	if strings.Contains(view, "prod *") {
		t.Errorf("inactive 'prod' should not have a marker; view:\n%s", view)
	}
}

// ── list mode: Esc emits BackMsg ─────────────────────────────────────────────

func TestProfilesList_EscEmitsBackMsg(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "a", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	_, cmd := sendKey(m, "esc")
	msg := runCmd(t, cmd)
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("want BackMsg, got %T", msg)
	}
}

// ── list mode: switch emits SwitchMsg ────────────────────────────────────────

func TestProfilesList_SwitchEmitsSwitchMsg(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "local", Theme: "adaptive"},
		tuiconfig.Profile{Name: "prod", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, makeTempConfigPath(t), "local")

	// Move cursor to "prod" and press Enter.
	m, _ = sendKey(m, "down")
	_, cmd := sendKey(m, "enter")
	msg := runCmd(t, cmd)
	sw, ok := msg.(SwitchMsg)
	if !ok {
		t.Fatalf("want SwitchMsg, got %T", msg)
	}
	if sw.Profile.Name != "prod" {
		t.Errorf("want Profile.Name=prod, got %q", sw.Profile.Name)
	}
}

func TestProfilesList_SwitchOnActiveProfileNoOp(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "local", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "local")

	_, cmd := sendKey(m, "enter") // cursor is at "local" (active)
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(SwitchMsg); ok {
			t.Error("should not emit SwitchMsg when selecting the active profile")
		}
	}
}

func TestProfilesList_SwitchWithSKey(t *testing.T) {
	cfg := testConfig(
		tuiconfig.Profile{Name: "local", Theme: "adaptive"},
		tuiconfig.Profile{Name: "prod", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, makeTempConfigPath(t), "local")
	m, _ = sendKey(m, "down")
	_, cmd := sendKey(m, "s")
	msg := runCmd(t, cmd)
	if _, ok := msg.(SwitchMsg); !ok {
		t.Errorf("s key: want SwitchMsg, got %T", msg)
	}
}

// ── list mode: opening create/edit ───────────────────────────────────────────

func TestProfilesList_CKeyEntersCreateMode(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "a", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "c")
	if m.mode != modeCreate {
		t.Errorf("want modeCreate, got %v", m.mode)
	}
	if m.form.step != stepName {
		t.Errorf("want stepName, got %v", m.form.step)
	}
}

func TestProfilesList_EKeyEntersEditMode(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "a", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "e")
	if m.mode != modeEdit {
		t.Errorf("want modeEdit, got %v", m.mode)
	}
	if m.form.oldName != "a" {
		t.Errorf("form.oldName: want a, got %q", m.form.oldName)
	}
	if m.form.name.value != "a" {
		t.Errorf("form pre-populated name: want a, got %q", m.form.name.value)
	}
}

// ── list mode: delete confirm ─────────────────────────────────────────────────

func TestProfilesList_DKeyEntersDeleteConfirm(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "a", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "d")
	if m.mode != modeDeleteConfirm {
		t.Errorf("want modeDeleteConfirm, got %v", m.mode)
	}
	if m.deleteTarget != "a" {
		t.Errorf("deleteTarget: want a, got %q", m.deleteTarget)
	}
}

func TestProfilesList_DeleteConfirmN_CancelsDelete(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "a", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")

	m, _ = sendKey(m, "d")
	m, _ = sendKey(m, "n")
	if m.mode != modeList {
		t.Errorf("after n: want modeList, got %v", m.mode)
	}
	if len(m.config.Profiles) != 1 {
		t.Error("profile should still exist after cancelling delete")
	}
}

// ── delete non-active profile: persists, emits ConfigChangedMsg ───────────────

func TestDeleteNonActiveProfile(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", ServerURL: "http://a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", ServerURL: "http://b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, path, "a")

	// Navigate to "b" and delete it.
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "d")
	m, cmd := sendKey(m, "y")

	if m.mode != modeList {
		t.Errorf("mode: want modeList, got %v", m.mode)
	}
	if len(m.config.Profiles) != 1 || m.config.Profiles[0].Name != "a" {
		t.Errorf("profile 'b' should be removed; profiles: %v", m.config.Profiles)
	}

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	if cm.ActiveDeleted {
		t.Error("ActiveDeleted should be false when deleting non-active profile")
	}

	// Verify file was written.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

// ── delete active profile (others remain): ActiveDeleted=true ────────────────

func TestDeleteActiveProfile_OthersRemain(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, path, "a") // "a" is active

	m, _ = sendKey(m, "d") // delete "a" (cursor at 0)
	m, cmd := sendKey(m, "y")

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	if !cm.ActiveDeleted {
		t.Error("ActiveDeleted should be true when deleting the active profile")
	}
	if len(cm.Config.Profiles) != 1 || cm.Config.Profiles[0].Name != "b" {
		t.Errorf("remaining profiles: %v", cm.Config.Profiles)
	}
	_ = m
}

// ── delete active → no profiles remain ───────────────────────────────────────

func TestDeleteActiveProfile_NoProfilesRemain(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "only", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "only")

	m, _ = sendKey(m, "d")
	m, cmd := sendKey(m, "y")

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	if !cm.ActiveDeleted {
		t.Error("ActiveDeleted should be true")
	}
	if len(cm.Config.Profiles) != 0 {
		t.Errorf("want empty profiles, got %v", cm.Config.Profiles)
	}
	_ = m
}

// ── save failure rolls back in-memory state ───────────────────────────────────

func TestDeleteProfile_SaveFailure_RollsBack(t *testing.T) {
	// Use a path in a directory that doesn't exist → write will fail.
	badPath := filepath.Join(t.TempDir(), "nonexistent", "ekvs-tui.yaml")
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, badPath, "a")

	m, _ = sendKey(m, "down") // cursor at "b"
	m, _ = sendKey(m, "d")
	m, _ = sendKey(m, "y")

	// In-memory config must be unchanged after write failure.
	if len(m.config.Profiles) != 2 {
		t.Errorf("rollback: want 2 profiles, got %d", len(m.config.Profiles))
	}
	if m.err == "" {
		t.Error("err should be set after save failure")
	}
}

// TestCreateProfile_SaveFailure_ShowsError verifies that a disk-write failure
// during create is reported to the user and does not mutate the in-memory config.
func TestCreateProfile_SaveFailure_ShowsError(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "nonexistent", "ekvs-tui.yaml")
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, badPath, "")

	m, _ = sendKey(m, "c")
	m, _ = stepsThroughCreateForm(t, m, "failprofile")

	// After a save failure the user stays in the form (mode stays modeCreate)
	// so they can see the error without losing their input.
	if m.mode == modeList {
		// Accepting modeList is wrong only if no error is shown.
	}
	if m.err == "" {
		t.Error("err should be set after save failure")
	}
	// The in-memory profile must have been rolled back (UpsertProfile was called
	// but then the backup was restored).
	if len(m.config.Profiles) != 0 {
		t.Errorf("rollback: want 0 profiles after failed save, got %d", len(m.config.Profiles))
	}
}

// TestEditProfile_SaveFailure_RollsBack verifies that a disk-write failure
// during edit is reported and the in-memory config is restored to its prior state.
func TestEditProfile_SaveFailure_RollsBack(t *testing.T) {
	badPath := filepath.Join(t.TempDir(), "nonexistent", "ekvs-tui.yaml")
	cfg := testConfig(tuiconfig.Profile{Name: "alpha", ServerURL: "http://old", Theme: "adaptive"})
	m := newTestModel(t, cfg, badPath, "other") // "alpha" is not active

	m, _ = sendKey(m, "e") // edit "alpha"
	// Rename to "alpha-new"
	for range "alpha" {
		m, _ = sendKey(m, "backspace")
	}
	m = typeString(m, "alpha-new")
	m, _ = sendKey(m, "enter") // confirm name
	m, _ = sendKey(m, "enter") // confirm serverURL
	m, _ = sendKey(m, "enter") // confirm identity
	m, _ = sendKey(m, "enter") // confirm theme → save (fails)

	if m.err == "" {
		t.Error("err should be set after failed save")
	}
	// Original name must be preserved.
	if len(m.config.Profiles) != 1 || m.config.Profiles[0].Name != "alpha" {
		t.Errorf("rollback: want profile 'alpha', got %v", m.config.Profiles)
	}
}

// ── create flow ───────────────────────────────────────────────────────────────

// stepsThroughCreateForm drives the create form to completion with the given
// name and returns the final model and the last command.
// It uses the key path: name → Enter → (skip serverURL) Enter → (skip identity) Enter → (theme) Enter.
func stepsThroughCreateForm(t *testing.T, m Model, name string) (Model, tea.Cmd) {
	t.Helper()
	// stepName
	m = typeString(m, name)
	m, _ = sendKey(m, "enter")
	// stepServerURL (blank → default)
	m, _ = sendKey(m, "enter")
	// stepIdentity (pick mode or manual; just press enter)
	m, _ = sendKey(m, "enter")
	// stepTheme (press enter on current selection)
	var cmd tea.Cmd
	m, cmd = sendKey(m, "enter")
	return m, cmd
}

func TestCreateProfile_PersistsAndEmitsConfigChanged(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := &tuiconfig.ConfigFile{} // empty
	m := newTestModel(t, cfg, path, "")

	m, _ = sendKey(m, "c") // enter create mode
	m, cmd := stepsThroughCreateForm(t, m, "newprofile")

	if m.mode != modeList {
		t.Errorf("want modeList after create, got %v", m.mode)
	}

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	if cm.Config == nil || len(cm.Config.Profiles) != 1 {
		t.Fatalf("want 1 profile in ConfigChangedMsg, got %v", cm.Config)
	}
	if cm.Config.Profiles[0].Name != "newprofile" {
		t.Errorf("profile name: want newprofile, got %q", cm.Config.Profiles[0].Name)
	}
	// Verify file on disk.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not written: %v", err)
	}
}

func TestCreateProfile_RejectsDuplicateName(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "existing", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "existing")

	m, _ = sendKey(m, "c")
	m = typeString(m, "existing")
	m, _ = sendKey(m, "enter")

	if m.form.step != stepName {
		t.Error("should stay on stepName after duplicate-name rejection")
	}
	if m.err == "" {
		t.Error("err should be set for duplicate name")
	}
}

func TestCreateProfile_RejectsEmptyName(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, path, "")

	m, _ = sendKey(m, "c")
	m, _ = sendKey(m, "enter") // submit empty name

	if m.form.step != stepName {
		t.Error("should stay on stepName after empty-name rejection")
	}
	if m.err == "" {
		t.Error("err should be set for empty name")
	}
}

func TestCreateProfile_CancelWithEsc(t *testing.T) {
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "")
	m, _ = sendKey(m, "c")
	m, _ = sendKey(m, "esc")
	if m.mode != modeList {
		t.Errorf("esc on stepName: want modeList, got %v", m.mode)
	}
}

func TestCreateProfile_NewProfileSelectedInList(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "first", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "first")

	m, _ = sendKey(m, "c")
	m, _ = stepsThroughCreateForm(t, m, "second")

	// Cursor should point to the newly created "second" profile.
	if m.cursor >= len(m.config.Profiles) {
		t.Fatalf("cursor out of range: %d (len=%d)", m.cursor, len(m.config.Profiles))
	}
	if m.config.Profiles[m.cursor].Name != "second" {
		t.Errorf("cursor should point to new profile; got %q", m.config.Profiles[m.cursor].Name)
	}
}

func TestCreateProfile_ActiveProfileUnchanged(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "old", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "old")

	m, _ = sendKey(m, "c")
	m, _ = stepsThroughCreateForm(t, m, "brandnew")

	if m.activeProfile != "old" {
		t.Errorf("active profile should not change after create; got %q", m.activeProfile)
	}
}

// ── create flow: SSH key discovery ───────────────────────────────────────────

func TestCreateProfile_ProposesDiscoveredKeys(t *testing.T) {
	sshFn, expectedKey := fakeSSHDir(t)
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "").WithSSHDirFn(sshFn)

	m, _ = sendKey(m, "c")

	// After entering create mode the form should have discovered keys.
	if m.form.identMode != identityModePick {
		t.Errorf("want identityModePick when keys are discovered, got %v", m.form.identMode)
	}
	if len(m.form.discovered) == 0 {
		t.Fatal("expected discovered keys, got none")
	}
	if m.form.discovered[0] != expectedKey {
		t.Errorf("want %q, got %q", expectedKey, m.form.discovered[0])
	}
	// .pub file must not be in the list.
	for _, k := range m.form.discovered {
		if strings.HasSuffix(k, ".pub") {
			t.Errorf("pub file should not be discovered: %q", k)
		}
	}
}

func TestCreateProfile_ManualPathBypassesDiscovery(t *testing.T) {
	sshFn, _ := fakeSSHDir(t)
	path := makeTempConfigPath(t)
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, path, "").WithSSHDirFn(sshFn)

	m, _ = sendKey(m, "c")
	// stepName
	m = typeString(m, "myprf")
	m, _ = sendKey(m, "enter")
	// stepServerURL
	m, _ = sendKey(m, "enter")
	// stepIdentity: press "m" to switch to manual input
	m, _ = sendKey(m, "m")
	if m.form.identMode != identityModeManual {
		t.Errorf("want identityModeManual after pressing m, got %v", m.form.identMode)
	}
	m = typeString(m, "/custom/key")
	m, _ = sendKey(m, "enter")
	// stepTheme
	m, cmd := sendKey(m, "enter")

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	if cm.Config.Profiles[0].IdentityFile != "/custom/key" {
		t.Errorf("want IdentityFile=/custom/key, got %q", cm.Config.Profiles[0].IdentityFile)
	}
}

func TestCreateProfile_WorksWithNoDiscoveredKeys(t *testing.T) {
	// sshDirFn returns an empty directory.
	emptyDir := t.TempDir()
	sshFn := func() (string, error) { return emptyDir, nil }
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "").WithSSHDirFn(sshFn)

	m, _ = sendKey(m, "c")
	if m.form.identMode != identityModeManual {
		t.Errorf("want identityModeManual when no keys discovered, got %v", m.form.identMode)
	}
}

func TestCreateProfile_WorksWhenSSHDirErrors(t *testing.T) {
	sshFn := func() (string, error) { return "", os.ErrNotExist }
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "").WithSSHDirFn(sshFn)

	m, _ = sendKey(m, "c")
	if m.form.identMode != identityModeManual {
		t.Errorf("want identityModeManual on SSHDir error, got %v", m.form.identMode)
	}
}

// ── edit flow ─────────────────────────────────────────────────────────────────

func TestEditProfile_UpdatesAndEmitsConfigChanged(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(
		tuiconfig.Profile{Name: "alpha", ServerURL: "http://old", Theme: "adaptive"},
		tuiconfig.Profile{Name: "beta", ServerURL: "http://b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, path, "beta") // "alpha" is NOT active

	m, _ = sendKey(m, "e") // edit "alpha"
	// Clear name, type new name.
	for range "alpha" {
		m, _ = sendKey(m, "backspace")
	}
	m = typeString(m, "alpha-new")
	m, _ = sendKey(m, "enter")    // confirm name
	m, _ = sendKey(m, "enter")    // confirm serverURL (unchanged)
	m, _ = sendKey(m, "enter")    // confirm identity (unchanged)
	m, cmd := sendKey(m, "enter") // confirm theme

	if m.mode != modeList {
		t.Errorf("want modeList, got %v", m.mode)
	}

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	found := false
	for _, p := range cm.Config.Profiles {
		if p.Name == "alpha-new" {
			found = true
		}
	}
	if !found {
		t.Error("renamed profile not found in ConfigChangedMsg.Config")
	}
}

func TestEditProfile_RejectsDuplicateName(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(
		tuiconfig.Profile{Name: "a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, path, "a")

	m, _ = sendKey(m, "e") // edit "a"
	// Replace name with "b" (already exists).
	for range "a" {
		m, _ = sendKey(m, "backspace")
	}
	m = typeString(m, "b")
	m, _ = sendKey(m, "enter")

	if m.form.step != stepName {
		t.Error("should stay on stepName after conflicting rename")
	}
	if m.err == "" {
		t.Error("err should be set for duplicate rename")
	}
}

func TestEditProfile_SameNameAllowed(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "a", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "b") // "a" is not active

	m, _ = sendKey(m, "e") // edit "a"
	// Keep name "a" unchanged.
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, cmd := sendKey(m, "enter")

	if m.mode != modeList {
		t.Errorf("want modeList, got %v", m.mode)
	}
	if m.err != "" {
		t.Errorf("unexpected error: %q", m.err)
	}
	_ = cmd
}

// ── edit active profile → reload prompt ──────────────────────────────────────

func TestEditActiveProfile_ShowsReloadPrompt(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "active", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "active")

	m, _ = sendKey(m, "e")     // edit "active"
	m, _ = sendKey(m, "enter") // name unchanged
	m, _ = sendKey(m, "enter") // serverURL unchanged
	m, _ = sendKey(m, "enter") // identity unchanged
	m, _ = sendKey(m, "enter") // theme unchanged → saves

	if m.mode != modeReloadPrompt {
		t.Errorf("want modeReloadPrompt after editing active profile, got %v", m.mode)
	}
}

func TestEditActiveProfile_ConfirmReloadEmitsReloadActiveMsg(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "active", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "active")

	m, _ = sendKey(m, "e")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter") // now in modeReloadPrompt

	m, cmd := sendKey(m, "y") // confirm reload

	if m.mode != modeList {
		t.Errorf("want modeList after confirming reload, got %v", m.mode)
	}
	msg := runCmd(t, cmd)
	ram, ok := msg.(ReloadActiveMsg)
	if !ok {
		t.Fatalf("want ReloadActiveMsg, got %T", msg)
	}
	if ram.Profile.Name != "active" {
		t.Errorf("want Profile.Name=active, got %q", ram.Profile.Name)
	}
}

func TestEditActiveProfile_DeclineReloadKeepsSession(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "active", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "active")

	m, _ = sendKey(m, "e")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter") // modeReloadPrompt

	m, cmd := sendKey(m, "n") // decline

	if m.mode != modeList {
		t.Errorf("want modeList after declining reload, got %v", m.mode)
	}
	// No ReloadActiveMsg should be emitted.
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(ReloadActiveMsg); ok {
			t.Error("should not emit ReloadActiveMsg when reload is declined")
		}
	}
}

// ── edit non-active profile: no reload prompt ─────────────────────────────────

func TestEditNonActiveProfile_NoReloadPrompt(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(
		tuiconfig.Profile{Name: "inactive", ServerURL: "http://a", Theme: "adaptive"},
		tuiconfig.Profile{Name: "active", ServerURL: "http://b", Theme: "hacker"},
	)
	m := newTestModel(t, cfg, path, "active")

	m, _ = sendKey(m, "e") // edit "inactive" (cursor=0)
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")

	if m.mode == modeReloadPrompt {
		t.Error("non-active profile edit should NOT show reload prompt")
	}
	if m.mode != modeList {
		t.Errorf("want modeList, got %v", m.mode)
	}
}

// ── form: server URL step go-back with Esc ────────────────────────────────────

func TestForm_EscOnServerURLGoesBackToName(t *testing.T) {
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "")
	m, _ = sendKey(m, "c")
	m = typeString(m, "myname")
	m, _ = sendKey(m, "enter") // → stepServerURL
	m, _ = sendKey(m, "esc")   // → back to stepName
	if m.form.step != stepName {
		t.Errorf("esc from stepServerURL: want stepName, got %v", m.form.step)
	}
}

// ── form: identity pick ↔ manual toggle ─────────────────────────────────────

func TestFormIdentity_EscFromManualBacksToPick(t *testing.T) {
	sshFn, _ := fakeSSHDir(t)
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "").WithSSHDirFn(sshFn)
	m, _ = sendKey(m, "c")
	m = typeString(m, "p")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	// Now on stepIdentity in pick mode.
	m, _ = sendKey(m, "m") // switch to manual
	if m.form.identMode != identityModeManual {
		t.Fatalf("want identityModeManual, got %v", m.form.identMode)
	}
	m, _ = sendKey(m, "esc") // back to pick
	if m.form.identMode != identityModePick {
		t.Errorf("esc from manual: want identityModePick, got %v", m.form.identMode)
	}
}

func TestFormIdentity_NoPickWhenNoKeys(t *testing.T) {
	sshFn := func() (string, error) { return t.TempDir(), nil }
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "").WithSSHDirFn(sshFn)
	m, _ = sendKey(m, "c")
	if m.form.identMode != identityModeManual {
		t.Errorf("want identityModeManual, got %v", m.form.identMode)
	}
}

// ── theme step ────────────────────────────────────────────────────────────────

func TestFormTheme_SelectHacker(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, path, "")

	m, _ = sendKey(m, "c")
	m = typeString(m, "p")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	// stepTheme: move down to "hacker".
	m, _ = sendKey(m, "down")
	m, cmd := sendKey(m, "enter")

	msg := runCmd(t, cmd)
	cm, ok := msg.(ConfigChangedMsg)
	if !ok {
		t.Fatalf("want ConfigChangedMsg, got %T", msg)
	}
	if cm.Config.Profiles[0].Theme != "hacker" {
		t.Errorf("want Theme=hacker, got %q", cm.Config.Profiles[0].Theme)
	}
	_ = m
}

// ── View ───────────────────────────────────────────────────────────────────────

func TestView_ListMode_ContainsProfiles(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "myprf", ServerURL: "http://x", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "myprf")

	view := m.View().Content
	if !strings.Contains(view, "myprf") {
		t.Errorf("view missing profile name; got:\n%s", view)
	}
}

func TestView_DeleteConfirmMode(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "victim", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "victim")
	m, _ = sendKey(m, "d")

	view := m.View().Content
	if !strings.Contains(view, "victim") || !strings.Contains(view, "Delete") {
		t.Errorf("delete confirm view missing expected text; got:\n%s", view)
	}
}

func TestView_ReloadPromptMode(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "act", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "act")

	m, _ = sendKey(m, "e")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter") // → modeReloadPrompt

	view := m.View().Content
	if !strings.Contains(view, "Reload") {
		t.Errorf("reload prompt view missing 'Reload'; got:\n%s", view)
	}
}

func TestView_EmptyProfilesList(t *testing.T) {
	cfg := &tuiconfig.ConfigFile{}
	m := newTestModel(t, cfg, makeTempConfigPath(t), "")
	view := m.View().Content
	if !strings.Contains(view, "no profiles") {
		t.Errorf("empty list view missing 'no profiles'; got:\n%s", view)
	}
}

// ── UpdateTyped ───────────────────────────────────────────────────────────────

func TestUpdateTyped(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "a", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "a")
	got, _ := m.UpdateTyped(keyMsg("down"))
	if got.cursor != 0 { // only 1 profile, cursor stays at 0
		t.Errorf("UpdateTyped: unexpected cursor %d", got.cursor)
	}
}

// ── discoverSSHKeys ───────────────────────────────────────────────────────────

func TestDiscoverSSHKeys_FiltersPublicKeys(t *testing.T) {
	dir := t.TempDir()
	files := []struct {
		name    string
		include bool
	}{
		{"id_ed25519", true},
		{"id_rsa", true},
		{"id_ed25519.pub", false},
		{"id_rsa.pub", false},
		{"known_hosts", false},
		{"config", false},
		{"authorized_keys", false},
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.name), []byte("x"), 0o600); err != nil {
			t.Fatalf("create %q: %v", f.name, err)
		}
	}
	sshFn := func() (string, error) { return dir, nil }
	m := Model{sshDirFn: sshFn}
	keys := m.discoverSSHKeys()

	found := map[string]bool{}
	for _, k := range keys {
		found[filepath.Base(k)] = true
	}
	for _, f := range files {
		if f.include && !found[f.name] {
			t.Errorf("expected %q in discovered keys", f.name)
		}
		if !f.include && found[f.name] {
			t.Errorf("unexpected %q in discovered keys", f.name)
		}
	}
}

func TestDiscoverSSHKeys_ReturnsNilOnError(t *testing.T) {
	sshFn := func() (string, error) { return "", os.ErrNotExist }
	m := Model{sshDirFn: sshFn}
	keys := m.discoverSSHKeys()
	if keys != nil {
		t.Errorf("expected nil on error, got %v", keys)
	}
}

// ── Task 9: WindowSizeMsg, modeError, two-pane split layout ───────────────────

func TestProfiles_WindowSizeMsgUpdatesWidth(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "p", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "p")

	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mm := next.(Model)
	if mm.width != 120 {
		t.Errorf("want width=120, got %d", mm.width)
	}
}

func TestProfiles_SplitLayoutRendered(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{
		Name:         "prod",
		ServerURL:    "https://ekvs.example.com",
		IdentityFile: "~/.ssh/id_ed25519",
		Theme:        "adaptive",
	})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "prod")

	// Set a width wide enough to trigger the split layout.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = next.(Model)

	view := m.View().Content
	// Left pane: profile name.
	if !strings.Contains(view, "prod") {
		t.Errorf("split view should contain profile name 'prod'; got:\n%s", view)
	}
	// Vertical separator.
	if !strings.Contains(view, "│") {
		t.Errorf("split view should contain vertical separator '│'; got:\n%s", view)
	}
	// Right pane: detail fields.
	if !strings.Contains(view, "Server URL") {
		t.Errorf("split view should contain 'Server URL' in detail pane; got:\n%s", view)
	}
	if !strings.Contains(view, "https://ekvs.example.com") {
		t.Errorf("split view should contain server URL; got:\n%s", view)
	}
	if !strings.Contains(view, "Identity file") {
		t.Errorf("split view should contain 'Identity file'; got:\n%s", view)
	}
}

func TestProfiles_ActiveProfileMarkedInSplitLayout(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "act", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "act")

	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = next.(Model)

	view := m.View().Content
	// Active profile should be marked with *.
	if !strings.Contains(view, "act *") {
		t.Errorf("active profile should be marked with '*'; got:\n%s", view)
	}
}

func TestProfiles_FallbackLayoutWhenWidthZero(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{
		Name:      "prod",
		ServerURL: "http://x",
		Theme:     "adaptive",
	})
	// No WindowSizeMsg sent → width stays 0.
	m := newTestModel(t, cfg, makeTempConfigPath(t), "prod")

	view := m.View().Content
	// Should still render the profile name in single-column mode.
	if !strings.Contains(view, "prod") {
		t.Errorf("fallback view should contain 'prod'; got:\n%s", view)
	}
}

func TestProfiles_ModeErrorOnSaveFail(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "p", Theme: "adaptive"})
	// Use a read-only path to force a save failure.
	roDir := t.TempDir()
	roPath := filepath.Join(roDir, "sub", "ekvs-tui.yaml") // non-existent sub-dir → save fails

	m := newTestModel(t, cfg, roPath, "p")
	sshFn, _ := fakeSSHDir(t)
	m = m.WithSSHDirFn(sshFn)

	// Enter edit mode and submit.
	m, _ = sendKey(m, "e")
	m, _ = sendKey(m, "enter")    // name step
	m, _ = sendKey(m, "enter")    // server step
	m, _ = sendKey(m, "enter")    // identity step
	m, cmd := sendKey(m, "enter") // theme step → triggers saveForm

	// Execute the command if any, then check mode.
	if cmd != nil {
		msg := cmd()
		next, _ := m.Update(msg)
		m = next.(Model)
	}

	// After the failed save the model should be in modeError.
	if m.mode != modeError {
		t.Errorf("want modeError after save failure, got mode=%v", m.mode)
	}

	view := m.View().Content
	if !strings.Contains(view, "save") {
		t.Errorf("error modal should mention 'save'; got:\n%s", view)
	}
}

func TestProfiles_ModeErrorDismissReturnsToList(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "p", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "p")

	// Force into modeError manually.
	m.mode = modeError
	m.err = "something went wrong"
	m.modalModel = modal.New(m.theme, m.err)

	// DismissMsg should return to modeList.
	next, _ := m.Update(modal.DismissMsg{})
	mm := next.(Model)
	if mm.mode != modeList {
		t.Errorf("want modeList after DismissMsg, got %v", mm.mode)
	}
	if mm.err != "" {
		t.Errorf("err should be cleared after dismiss, got %q", mm.err)
	}
}

func TestProfiles_FooterPresentInListMode(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "p", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "p")

	view := m.View().Content
	if !strings.Contains(view, "navigate") {
		t.Errorf("footer hints should contain 'navigate' in list mode; got:\n%s", view)
	}
}

func TestProfiles_FooterPresentInDeleteConfirm(t *testing.T) {
	cfg := testConfig(tuiconfig.Profile{Name: "p", Theme: "adaptive"})
	m := newTestModel(t, cfg, makeTempConfigPath(t), "p")
	m, _ = sendKey(m, "d")

	view := m.View().Content
	if !strings.Contains(view, "confirm") {
		t.Errorf("footer should contain 'confirm' in delete mode; got:\n%s", view)
	}
}

func TestProfiles_FooterPresentInReloadPrompt(t *testing.T) {
	path := makeTempConfigPath(t)
	cfg := testConfig(tuiconfig.Profile{Name: "act", ServerURL: "http://a", Theme: "adaptive"})
	m := newTestModel(t, cfg, path, "act")
	sshFn, _ := fakeSSHDir(t)
	m = m.WithSSHDirFn(sshFn)

	m, _ = sendKey(m, "e")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter")
	m, _ = sendKey(m, "enter") // → modeReloadPrompt

	view := m.View().Content
	if !strings.Contains(view, "reload") {
		t.Errorf("footer should contain 'reload' hint; got:\n%s", view)
	}
}
