package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/client"
	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/session"
	"ekvs/internal/tui/spinner"
	"ekvs/internal/tui/theme"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "internal", "ssh", "testdata")
	abs, _ := filepath.Abs(root)
	return abs
}

func testSession(t *testing.T) *session.Session {
	t.Helper()
	pem, err := os.ReadFile(filepath.Join(testdataDir(), "ed25519"))
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pem)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	sess := &session.Session{}
	if err := sess.SetAuthenticated(signer, pub, internalssh.Fingerprint(pub)); err != nil {
		t.Fatalf("SetAuthenticated: %v", err)
	}
	return sess
}

// fakeClient implements apiClient for tests.
type fakeClient struct {
	secrets      []client.SecretEntry
	listErr      error
	setErr       error
	deleteErr    error
	setCalled    [2]string // [key, value]
	deleteCalled string
}

func (f *fakeClient) ListSecrets(_ string) ([]client.SecretEntry, error) {
	return f.secrets, f.listErr
}
func (f *fakeClient) SetSecret(_, key, value string) error {
	f.setCalled = [2]string{key, value}
	return f.setErr
}
func (f *fakeClient) DeleteSecret(_, key string) error {
	f.deleteCalled = key
	return f.deleteErr
}

func newTestModel(t *testing.T, secrets []client.SecretEntry) Model {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{secrets: secrets}
	sess := testSession(t)
	return newWithClient("testproject", fc, sess, th)
}

// keyMsg builds a tea.KeyPressMsg.
func keyMsg(key string) tea.KeyPressMsg {
	switch key {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
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

func applyFetched(m Model, secrets []client.SecretEntry) Model {
	next, _ := m.Update(FetchedMsg{Secrets: secrets})
	mm, _ := next.(Model)
	return mm
}

// encryptBlob encrypts plaintext using the session (used to build test entries).
func encryptBlob(t *testing.T, sess *session.Session, plaintext string) string {
	t.Helper()
	blob, err := sess.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	return blob
}

// ── FetchedMsg ────────────────────────────────────────────────────────────────

func TestSecretsModel_FetchUpdates(t *testing.T) {
	m := newTestModel(t, nil)
	sess := testSession(t)
	blob := encryptBlob(t, sess, "myvalue")

	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	if len(m.secrets) != 1 {
		t.Errorf("want 1 secret, got %d", len(m.secrets))
	}
	if m.loading {
		t.Error("loading should be false after FetchedMsg")
	}
	if m.err != nil {
		t.Errorf("err should be nil, got %v", m.err)
	}
}

// ── ErrMsg ────────────────────────────────────────────────────────────────────

func TestSecretsModel_ErrDisplayed(t *testing.T) {
	m := newTestModel(t, nil)
	next, _ := m.Update(ErrMsg{Err: errors.New("boom")})
	mm := next.(Model)

	if mm.err == nil {
		t.Fatal("expected err to be set")
	}
	view := mm.View().Content
	if !strings.Contains(view, "boom") {
		t.Errorf("view should contain error text, got:\n%s", view)
	}
}

func TestSecretsModel_ErrClearedOnModalDismiss(t *testing.T) {
	m := newTestModel(t, nil)
	next, _ := m.Update(ErrMsg{Err: errors.New("boom")})
	m = next.(Model)

	if m.mode != modeError {
		t.Fatalf("want modeError after ErrMsg, got %v", m.mode)
	}
	if m.err == nil {
		t.Fatal("err should be set after ErrMsg")
	}

	// Dismiss modal with Enter → err should be cleared.
	m2, cmd := m.UpdateTyped(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		if dmsg, ok := cmd().(modal.DismissMsg); ok {
			m2, _ = m2.UpdateTyped(dmsg)
		}
	}
	if m2.mode != modeList {
		t.Errorf("want modeList after dismiss, got %v", m2.mode)
	}
	if m2.err != nil {
		t.Errorf("err should be cleared after modal dismiss, got %v", m2.err)
	}
}

// ── Cursor movement ───────────────────────────────────────────────────────────

func TestSecretsModel_CursorDown(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	m := newTestModel(t, nil)
	m = applyFetched(m, []client.SecretEntry{
		{Key: "a", Value: blob},
		{Key: "b", Value: blob},
	})

	m, _ = sendKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("want cursor=1, got %d", m.cursor)
	}
}

func TestSecretsModel_CursorUp(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	m := newTestModel(t, nil)
	m = applyFetched(m, []client.SecretEntry{
		{Key: "a", Value: blob},
		{Key: "b", Value: blob},
	})

	m, _ = sendKey(m, "up")
	if m.cursor != 1 {
		t.Errorf("want cursor wrapped to 1 (last), got %d", m.cursor)
	}
}

// ── Pagination ────────────────────────────────────────────────────────────────

func TestSecretsModel_Pagination(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	entries := make([]client.SecretEntry, 15)
	for i := range entries {
		entries[i] = client.SecretEntry{Key: "k", Value: blob}
	}

	m := newTestModel(t, nil)
	m = applyFetched(m, entries)

	if len(m.pageSecrets()) != 10 {
		t.Errorf("want 10 items on page 0, got %d", len(m.pageSecrets()))
	}

	m, _ = sendKey(m, "right")
	if m.page != 1 {
		t.Errorf("want page=1, got %d", m.page)
	}
	if len(m.pageSecrets()) != 5 {
		t.Errorf("want 5 items on page 1, got %d", len(m.pageSecrets()))
	}
}

// ── BackMsg on Esc ────────────────────────────────────────────────────────────

func TestSecretsModel_BackOnEsc(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, nil)

	_, cmd := sendKey(m, "esc")
	if cmd == nil {
		t.Fatal("want non-nil cmd on esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("want BackMsg, got %T", msg)
	}
}

// ── modeAdd ───────────────────────────────────────────────────────────────────

func TestSecretsModel_AddMode(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, nil)

	m, _ = sendKey(m, "n")
	if m.mode != modeAdd {
		t.Fatalf("want modeAdd, got %v", m.mode)
	}
	if m.activeField != 0 {
		t.Errorf("want activeField=0 (key), got %d", m.activeField)
	}

	// Type into key field.
	for _, ch := range "mykey" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	if m.inputKey != "mykey" {
		t.Errorf("want inputKey=mykey, got %q", m.inputKey)
	}

	// Tab advances to value field.
	m, _ = sendKey(m, "tab")
	if m.activeField != 1 {
		t.Errorf("want activeField=1 (value), got %d", m.activeField)
	}
}

func TestSecretsModel_AddModeEscCancels(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, nil)

	m, _ = sendKey(m, "n")
	m.inputKey = "partial"
	m, _ = sendKey(m, "esc")

	if m.mode != modeList {
		t.Errorf("want modeList after esc, got %v", m.mode)
	}
	if m.inputKey != "" {
		t.Errorf("want inputKey cleared, got %q", m.inputKey)
	}
}

func TestSecretsModel_AddModeSubmit(t *testing.T) {
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{secrets: []client.SecretEntry{}}
	sess := testSession(t)
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{})

	m, _ = sendKey(m, "n")
	// Type key
	for _, ch := range "newkey" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	m, _ = sendKey(m, "tab") // advance to value
	// Type value
	for _, ch := range "myvalue" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}

	_, cmd := sendKey(m, "enter")
	if cmd == nil {
		t.Fatal("want non-nil cmd after submit")
	}
	result := cmd()
	// Should produce FetchedMsg (set succeeded → list re-fetched).
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("want FetchedMsg, got %T: %v", result, result)
	}
	// Verify SetSecret was called.
	if fc.setCalled[0] != "newkey" {
		t.Errorf("want SetSecret key=newkey, got %q", fc.setCalled[0])
	}
	// Value should be an encrypted blob (not plaintext).
	if fc.setCalled[1] == "myvalue" {
		t.Error("SetSecret should receive encrypted blob, not plaintext")
	}
	if fc.setCalled[1] == "" {
		t.Error("SetSecret value should not be empty")
	}
}

// ── modeEdit ──────────────────────────────────────────────────────────────────

func TestSecretsModel_EditMode(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "originalvalue")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	m, _ = sendKey(m, "e")
	if m.mode != modeEdit {
		t.Fatalf("want modeEdit, got %v", m.mode)
	}
	if m.inputKey != "k" {
		t.Errorf("want inputKey=k, got %q", m.inputKey)
	}
	if m.inputValue != "originalvalue" {
		t.Errorf("want inputValue=originalvalue, got %q", m.inputValue)
	}
}

func TestSecretsModel_EditModeSubmit(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "old")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{secrets: []client.SecretEntry{{Key: "k", Value: blob}}}
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	m, _ = sendKey(m, "e")
	// Clear value field with backspace and type new value.
	for range "old" {
		m, _ = sendKey(m, "backspace")
	}
	for _, ch := range "newval" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}

	_, cmd := sendKey(m, "enter")
	if cmd == nil {
		t.Fatal("want non-nil cmd")
	}
	result := cmd()
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("want FetchedMsg, got %T", result)
	}
	if fc.setCalled[0] != "k" {
		t.Errorf("want SetSecret key=k, got %q", fc.setCalled[0])
	}
}

// ── modeDelete ────────────────────────────────────────────────────────────────

func TestSecretsModel_DeleteMode(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	m := newTestModel(t, nil)
	m = applyFetched(m, []client.SecretEntry{{Key: "todelete", Value: blob}})

	m, _ = sendKey(m, "d")
	if m.mode != modeDelete {
		t.Fatalf("want modeDelete, got %v", m.mode)
	}
	view := m.View().Content
	if !strings.Contains(view, "todelete") {
		t.Errorf("view should show key in confirm prompt, got:\n%s", view)
	}
}

func TestSecretsModel_DeleteConfirm(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{secrets: []client.SecretEntry{}}
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "todelete", Value: blob}})

	m, _ = sendKey(m, "d")
	_, cmd := sendKey(m, "y")
	if cmd == nil {
		t.Fatal("want non-nil cmd")
	}
	result := cmd()
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("want FetchedMsg, got %T", result)
	}
	if fc.deleteCalled != "todelete" {
		t.Errorf("want DeleteSecret called with todelete, got %q", fc.deleteCalled)
	}
}

func TestSecretsModel_DeleteCancel(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	m, _ = sendKey(m, "d")
	m, _ = sendKey(m, "n")
	if m.mode != modeList {
		t.Errorf("want modeList after cancel, got %v", m.mode)
	}
	if fc.deleteCalled != "" {
		t.Errorf("DeleteSecret should not be called, got %q", fc.deleteCalled)
	}
}

// ── View decryption ───────────────────────────────────────────────────────────

func TestSecretsModel_ViewDecryptsValues(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "supersecret")
	m := newTestModel(t, nil)
	// Replace session with the one that has the enc key.
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m = newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	view := m.View().Content
	if !strings.Contains(view, "supersecret") {
		t.Errorf("view should contain decrypted value 'supersecret', got:\n%s", view)
	}
	if strings.Contains(view, blob) {
		t.Errorf("view should not contain raw encrypted blob, got:\n%s", view)
	}
}

func TestSecretsModel_ViewShowsErrorOnDecryptFail(t *testing.T) {
	// Use an unauthenticated session so Decrypt returns ErrNotAuthenticated.
	th, _ := theme.NewTheme("adaptive")
	unauthSess := &session.Session{} // not authenticated
	fc := &fakeClient{}
	m := newWithClient("proj", fc, unauthSess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: "notablob"}})

	view := m.View().Content
	if !strings.Contains(view, "<error>") {
		t.Errorf("view should show <error> for failed decryption, got:\n%s", view)
	}
}

// ── Spinner ───────────────────────────────────────────────────────────────────

func TestSecretsModel_SpinnerTickForwarded(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	frameBefore := m.spinner.View()
	m2, cmd := m.UpdateTyped(spinner.TickMsg{})
	if m2.spinner.View() == frameBefore {
		t.Error("spinner frame should advance on TickMsg")
	}
	if cmd == nil {
		t.Error("Update(TickMsg) should return next tick cmd")
	}
}

func TestSecretsModel_LoadingViewContainsSpinner(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	view := m.View().Content
	if !strings.Contains(view, "Loading") {
		t.Errorf("loading view should contain 'Loading', got:\n%s", view)
	}
}

func TestSecretsModel_InitReturnsBatchCmd(t *testing.T) {
	m := newTestModel(t, nil)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil cmd")
	}
	// Fetch path works independently.
	result := m.fetchCmd()()
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("fetchCmd should return FetchedMsg, got %T", result)
	}
}

// ── Footer ────────────────────────────────────────────────────────────────────

func TestSecretsModel_FooterShowsHints(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	m := newTestModel(t, nil)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	view := m.View().Content
	if !strings.Contains(view, "navigate") {
		t.Errorf("footer should contain navigation hints, got:\n%s", view)
	}
}

func TestSecretsModel_FooterHintsChangeByMode(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, nil)

	// Add mode, field 0 (key): footer shows "next field".
	m, _ = sendKey(m, "n")
	view := m.View().Content
	if !strings.Contains(view, "next field") {
		t.Errorf("add mode field-0 footer should contain 'next field', got:\n%s", view)
	}

	// Advance to field 1 (value): footer shows "confirm".
	for _, ch := range "k" {
		m, _ = m.UpdateTyped(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	m, _ = sendKey(m, "tab")
	view2 := m.View().Content
	if !strings.Contains(view2, "confirm") {
		t.Errorf("add mode field-1 footer should contain 'confirm', got:\n%s", view2)
	}
}

// ── Error modal ───────────────────────────────────────────────────────────────

func TestSecretsModel_ErrSwitchesToModalMode(t *testing.T) {
	m := newTestModel(t, nil)
	next, _ := m.Update(ErrMsg{Err: errors.New("boom")})
	mm := next.(Model)

	if mm.mode != modeError {
		t.Errorf("want modeError after ErrMsg, got %v", mm.mode)
	}
}

func TestSecretsModel_ModalViewContainsError(t *testing.T) {
	m := newTestModel(t, nil)
	next, _ := m.Update(ErrMsg{Err: errors.New("network timeout")})
	mm := next.(Model)

	view := mm.View().Content
	if !strings.Contains(view, "network timeout") {
		t.Errorf("modal view should contain error message, got:\n%s", view)
	}
}

// ── Tabular display ───────────────────────────────────────────────────────────

func TestSecretsModel_TableHeaderPresent(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "value1")
	m := newTestModel(t, nil)
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m = newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{
		{Key: "mykey", Value: blob},
	})

	view := m.View().Content
	if !strings.Contains(view, "KEY") {
		t.Errorf("table view should contain 'KEY' header, got:\n%s", view)
	}
	if !strings.Contains(view, "VALUE") {
		t.Errorf("table view should contain 'VALUE' header, got:\n%s", view)
	}
}

func TestSecretsModel_TableSeparatorPresent(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{{Key: "k", Value: blob}})

	view := m.View().Content
	// The bubbles/v2 table component renders a header row and data rows
	// side-by-side using lipgloss. Both KEY and VALUE columns must appear.
	if !strings.Contains(view, "KEY") {
		t.Errorf("table view should contain 'KEY' header, got:\n%s", view)
	}
	if !strings.Contains(view, "VALUE") {
		t.Errorf("table view should contain 'VALUE' header, got:\n%s", view)
	}
}

func TestSecretsModel_TableRowsAligned(t *testing.T) {
	sess := testSession(t)
	blob1 := encryptBlob(t, sess, "val1")
	blob2 := encryptBlob(t, sess, "val2")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m := newWithClient("proj", fc, sess, th)
	m = applyFetched(m, []client.SecretEntry{
		{Key: "short", Value: blob1},
		{Key: "a-much-longer-key", Value: blob2},
	})

	view := m.View().Content
	// Both keys must appear in the rendered output.
	if !strings.Contains(view, "short") {
		t.Errorf("view should contain key 'short', got:\n%s", view)
	}
	if !strings.Contains(view, "a-much-longer-key") {
		t.Errorf("view should contain key 'a-much-longer-key', got:\n%s", view)
	}
	// Both values must be decrypted and visible.
	if !strings.Contains(view, "val1") {
		t.Errorf("view should contain decrypted value 'val1', got:\n%s", view)
	}
	if !strings.Contains(view, "val2") {
		t.Errorf("view should contain decrypted value 'val2', got:\n%s", view)
	}
}

// ── Task 8: modeSearch ────────────────────────────────────────────────────────

// buildSearchModel creates a model pre-loaded with three secrets with non-overlapping prefixes.
func buildSearchModel(t *testing.T) Model {
	t.Helper()
	sess := testSession(t)
	b1 := encryptBlob(t, sess, "v1")
	b2 := encryptBlob(t, sess, "v2")
	b3 := encryptBlob(t, sess, "v3")
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{}
	m := newWithClient("proj", fc, sess, th)
	return applyFetched(m, []client.SecretEntry{
		{Key: "apple", Value: b1},
		{Key: "banana", Value: b2},
		{Key: "cherry", Value: b3},
	})
}

func TestSecretsModel_SlashEntersModeSearch(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	if m.mode != modeSearch {
		t.Errorf("want modeSearch after '/', got %v", m.mode)
	}
	if m.searchQuery != "" {
		t.Errorf("searchQuery should be empty on entry, got %q", m.searchQuery)
	}
}

func TestSecretsModel_SearchFiltersRows(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "n")

	filtered := m.filteredSecrets()
	if len(filtered) != 1 {
		t.Fatalf("want 1 filtered secret for query 'ban', got %d", len(filtered))
	}
	if filtered[0].Key != "banana" {
		t.Errorf("want 'banana', got %q", filtered[0].Key)
	}
}

func TestSecretsModel_SearchCaseInsensitive(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "A")
	m, _ = sendKey(m, "P")

	filtered := m.filteredSecrets()
	// only "apple" contains "ap"
	if len(filtered) != 1 {
		t.Fatalf("want 1 result for 'AP' (case-insensitive), got %d", len(filtered))
	}
	if filtered[0].Key != "apple" {
		t.Errorf("want 'apple', got %q", filtered[0].Key)
	}
}

func TestSecretsModel_SearchCursorResetOnKeystroke(t *testing.T) {
	m := buildSearchModel(t)
	// Move cursor to position 2 first.
	m, _ = sendKey(m, "down")
	m, _ = sendKey(m, "down")
	if m.cursor != 2 {
		t.Fatalf("setup: want cursor=2, got %d", m.cursor)
	}
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "a")
	if m.cursor != 0 {
		t.Errorf("cursor should reset to 0 on filter change, got %d", m.cursor)
	}
}

func TestSecretsModel_SearchBackspace(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "backspace")

	if m.searchQuery != "b" {
		t.Errorf("want searchQuery='b' after backspace, got %q", m.searchQuery)
	}
}

func TestSecretsModel_SearchEnterReturnToListKeepsFilter(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "n")
	m, _ = sendKey(m, "enter")

	if m.mode != modeList {
		t.Errorf("want modeList after Enter, got %v", m.mode)
	}
	if m.searchQuery != "ban" {
		t.Errorf("filter should be kept after Enter, got %q", m.searchQuery)
	}
	filtered := m.filteredSecrets()
	if len(filtered) != 1 {
		t.Errorf("filter should still be active, got %d results", len(filtered))
	}
}

func TestSecretsModel_SearchEscReturnToListKeepsFilter(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "n")
	m, _ = sendKey(m, "esc")

	if m.mode != modeList {
		t.Errorf("want modeList after Esc, got %v", m.mode)
	}
	if m.searchQuery != "ban" {
		t.Errorf("filter should be kept after Esc from search, got %q", m.searchQuery)
	}
}

func TestSecretsModel_DoubleEscClearsFilter(t *testing.T) {
	m := buildSearchModel(t)
	// Enter search, type "ban", confirm with Esc → back to modeList with filter active.
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "n")
	m, _ = sendKey(m, "esc")
	if m.searchQuery == "" {
		t.Fatalf("filter should be active before double-esc")
	}
	// Second Esc in modeList with non-empty filter → clears filter.
	m, _ = sendKey(m, "esc")
	if m.searchQuery != "" {
		t.Errorf("searchQuery should be cleared after second Esc, got %q", m.searchQuery)
	}
	if len(m.filteredSecrets()) != 3 {
		t.Errorf("all secrets should be visible after clear, got %d", len(m.filteredSecrets()))
	}
}

func TestSecretsModel_SearchViewShowsFilteredResults(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "n")
	m, _ = sendKey(m, "enter")

	view := m.View().Content
	if !strings.Contains(view, "banana") {
		t.Errorf("view should contain 'banana', got:\n%s", view)
	}
	// Only "banana" matches "ban".
	if len(m.pageSecrets()) != 1 {
		t.Errorf("only 1 row should be shown, got %d", len(m.pageSecrets()))
	}
}

func TestSecretsModel_SearchNoMatchShowsEmptyMessage(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "z")
	m, _ = sendKey(m, "z")
	m, _ = sendKey(m, "z")
	m, _ = sendKey(m, "enter")

	view := m.View().Content
	if !strings.Contains(view, "zz") {
		t.Errorf("view should mention the query in empty message, got:\n%s", view)
	}
}

func TestSecretsModel_SearchFooterHint(t *testing.T) {
	m := buildSearchModel(t)
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "b")
	m, _ = sendKey(m, "a")

	view := m.View().Content
	if !strings.Contains(view, "Search:") {
		t.Errorf("footer should show 'Search:' hint in modeSearch, got:\n%s", view)
	}
}
