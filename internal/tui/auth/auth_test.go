package auth

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
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
		if mm.state != statePrompt {
			t.Errorf("expected statePrompt, got %v", mm.state)
		}
	}
}

func TestAuth_MissingKey_TransitionsToError(t *testing.T) {
	m := newModel(t, "/nonexistent/key")
	cmd := m.Init()
	msg := runCmd(cmd)
	next, _ := m.Update(msg)
	if mm, ok := next.(Model); ok {
		if mm.state != stateError {
			t.Errorf("expected stateError, got %v", mm.state)
		}
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

func TestAuth_CorrectPassphrase_EmitsSuccess(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	// Type "testpass"
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

func TestAuth_WrongPassphrase_TransitionsToError(t *testing.T) {
	m := newModel(t, "../../../internal/ssh/testdata/ed25519-passphrase")
	for _, ch := range "wrongpass" {
		next, _ := m.Update(tea.KeyPressMsg{Code: rune(ch), Text: string(ch)})
		if mm, ok := next.(Model); ok {
			m = mm
		}
	}
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if mm, ok := next.(Model); ok {
		if mm.state != stateError {
			t.Errorf("expected stateError after wrong passphrase, got %v", mm.state)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
