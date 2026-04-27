package auth

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/theme"
)

func newModel(t *testing.T, keyFile string) Model {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	return New(keyFile, th)
}

// runCmd executes a tea.Cmd and returns the resulting message, or nil.
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestAuth_UnencryptedKey_EmitsSuccessImmediately(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519")
	cmd := m.Init()
	msg := runCmd(cmd)
	if _, ok := msg.(tryLoadMsg); !ok {
		t.Fatalf("expected tryLoadMsg from Init, got %T", msg)
	}
	next, cmd2 := m.Update(msg)
	outMsg := runCmd(cmd2)
	if _, ok := outMsg.(AuthSuccessMsg); !ok {
		_ = next
		t.Fatalf("expected AuthSuccessMsg, got %T", outMsg)
	}
}

func TestAuth_PassphraseKey_NoSuccessOnInit(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	cmd := m.Init()
	msg := runCmd(cmd)
	next, cmd2 := m.Update(msg)
	outMsg := runCmd(cmd2)
	if _, ok := outMsg.(AuthSuccessMsg); ok {
		t.Fatal("unexpected AuthSuccessMsg for passphrase-protected key")
	}
	if mm, ok := next.(Model); ok {
		if mm.showModal {
			t.Error("modal should not be shown for a passphrase-protected key (prompt expected)")
		}
	}
}

func TestAuth_MissingKey_ShowsModal(t *testing.T) {
	m := newModel(t, "/nonexistent/key")
	cmd := m.Init()
	msg := runCmd(cmd)
	next, _ := m.Update(msg)
	mm := next.(Model)
	if !mm.showModal {
		t.Error("expected modal to be shown for missing key")
	}
	view := mm.View().Content
	if !strings.Contains(view, "not found") {
		t.Errorf("modal should mention 'not found'; got:\n%s", view)
	}
}

func TestAuth_MissingKey_ModalDismiss_AllowsRetry(t *testing.T) {
	m := newModel(t, "/nonexistent/key")
	msg := runCmd(m.Init())
	next, _ := m.Update(msg)
	m = next.(Model)
	if !m.showModal {
		t.Fatal("precondition: modal should be showing")
	}
	// Dismiss the modal.
	next, _ = m.Update(modal.DismissMsg{})
	m = next.(Model)
	if m.showModal {
		t.Error("modal should be dismissed after DismissMsg")
	}
	// The passphrase prompt should be visible again.
	view := m.View().Content
	if !strings.Contains(view, "passphrase") {
		t.Errorf("prompt should be visible after dismissal; got:\n%s", view)
	}
}

func TestAuth_EscEmitsCancelMsg(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	msg := runCmd(cmd)
	if _, ok := msg.(AuthCancelMsg); !ok {
		t.Fatalf("expected AuthCancelMsg, got %T", msg)
	}
}

func TestAuth_EscIgnoredWhenModalShowing(t *testing.T) {
	// When the modal is active, Esc should dismiss the modal (not cancel auth).
	m := newModel(t, "/nonexistent/key")
	msg := runCmd(m.Init())
	next, _ := m.Update(msg)
	m = next.(Model)
	if !m.showModal {
		t.Fatal("precondition: modal should be showing")
	}
	// Esc inside the modal should trigger DismissMsg, not AuthCancelMsg.
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		outMsg := cmd()
		if _, ok := outMsg.(AuthCancelMsg); ok {
			t.Error("Esc while modal is showing should not emit AuthCancelMsg")
		}
	}
}

func TestAuth_CorrectPassphrase_EmitsSuccess(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	for _, ch := range "testpass" {
		next, _ := m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		if mm, ok := next.(Model); ok {
			m = mm
		}
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := runCmd(cmd)
	if _, ok := msg.(AuthSuccessMsg); !ok {
		t.Fatalf("expected AuthSuccessMsg, got %T", msg)
	}
}

func TestAuth_WrongPassphrase_ShowsModal(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	for _, ch := range "wrongpass" {
		next, _ := m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		if mm, ok := next.(Model); ok {
			m = mm
		}
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := next.(Model)
	if !mm.showModal {
		t.Error("expected modal to be shown after wrong passphrase")
	}
	view := mm.View().Content
	if !strings.Contains(view, "passphrase") {
		t.Errorf("modal should mention 'passphrase'; got:\n%s", view)
	}
}

func TestAuth_WrongPassphrase_DismissAndRetry(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	for _, ch := range "wrongpass" {
		next, _ := m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		m = next.(Model)
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = next.(Model)
	if !m.showModal {
		t.Fatal("precondition: modal should be showing after wrong passphrase")
	}
	// Dismiss modal → back to prompt with cleared input.
	next, _ = m.Update(modal.DismissMsg{})
	m = next.(Model)
	if m.showModal {
		t.Error("modal should be dismissed")
	}
	if m.input != "" {
		t.Errorf("input should be cleared after dismiss, got %q", m.input)
	}
	// Now type the correct passphrase and succeed.
	for _, ch := range "testpass" {
		next, _ = m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		m = next.(Model)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	msg := runCmd(cmd)
	if _, ok := msg.(AuthSuccessMsg); !ok {
		t.Fatalf("expected AuthSuccessMsg after retry, got %T", msg)
	}
}

func TestAuth_ViewShowsPassphrasePrompt(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	view := m.View().Content
	if !strings.Contains(view, "passphrase") {
		t.Errorf("view should show passphrase prompt; got:\n%s", view)
	}
	if !strings.Contains(view, "Enter") {
		t.Errorf("view should show hint with 'Enter'; got:\n%s", view)
	}
}

func TestAuth_ViewMasksInput(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	for _, ch := range "abc" {
		next, _ := m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		m = next.(Model)
	}
	view := m.View().Content
	if strings.Contains(view, "abc") {
		t.Error("view should not show plaintext input; it should be masked with *")
	}
	if !strings.Contains(view, "***") {
		t.Errorf("view should show three * for three input chars; got:\n%s", view)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
