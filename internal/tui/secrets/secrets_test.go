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
	"ekvs/internal/tui/session"
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

func TestSecretsModel_ErrClearedOnKeypress(t *testing.T) {
	m := newTestModel(t, nil)
	next, _ := m.Update(ErrMsg{Err: errors.New("boom")})
	m = next.(Model)

	m, _ = sendKey(m, "esc")
	// esc emits BackMsg but clears err first
	// use a neutral key that doesn't navigate away
	m2 := newTestModel(t, nil)
	next2, _ := m2.Update(ErrMsg{Err: errors.New("boom")})
	m2 = next2.(Model)
	m2, _ = sendKey(m2, "n") // switches to modeAdd but clears err
	if m2.err != nil {
		t.Errorf("err should be cleared on keypress, got %v", m2.err)
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
