package auth

// auth_e2e_test.go — end-to-end user-journey tests for the Auth screen.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/modal"
)

const (
	e2eKeyUnencrypted = "../../../internal/ssh/testdata/ed25519"
	e2eKeyPassphrase  = "../../../internal/ssh/testdata/ed25519-passphrase"
	e2eCorrectPass    = "testpass"
	e2eWrongPass      = "wrongpass"
)

// TestE2E_Auth_ViewShowsPassphrasePrompt verifies the initial view before Init.
func TestE2E_Auth_ViewShowsPassphrasePrompt(t *testing.T) {
	m := newModel(t, e2eKeyPassphrase)
	view := m.View().Content
	if !strings.Contains(strings.ToLower(view), "passphrase") {
		t.Errorf("initial view should contain 'passphrase';\n%s", view)
	}
}

// TestE2E_Auth_UnencryptedKeySucceedsWithoutPassphrase tests the full flow for
// an unencrypted key: Init → tryLoadMsg → AuthSuccessMsg emitted.
func TestE2E_Auth_UnencryptedKeySucceedsWithoutPassphrase(t *testing.T) {
	m := newModel(t, e2eKeyUnencrypted)
	initCmd := m.Init()
	if initCmd == nil {
		t.Fatal("Init should return a non-nil cmd")
	}
	tryMsg := runCmd(initCmd)
	next, authCmd := m.Update(tryMsg)
	successMsg := runCmd(authCmd)
	_ = next
	if _, ok := successMsg.(AuthSuccessMsg); !ok {
		t.Fatalf("expected AuthSuccessMsg for unencrypted key, got %T", successMsg)
	}
}

// TestE2E_Auth_WrongThenCorrectPassphrase simulates the full journey:
// init → prompt → wrong passphrase → modal → dismiss → correct passphrase → success.
func TestE2E_Auth_WrongThenCorrectPassphrase(t *testing.T) {
	m := newModel(t, e2eKeyPassphrase)

	// Init: key needs passphrase, show prompt.
	tryMsg := runCmd(m.Init())
	m2, _ := m.Update(tryMsg)
	m = m2.(Model)
	if m.showModal {
		t.Fatal("modal should not be shown on passphrase prompt")
	}

	// Type wrong passphrase and submit.
	for _, ch := range e2eWrongPass {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	m2, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = m2.(Model)
	if !m.showModal {
		t.Fatal("modal should be visible after wrong passphrase")
	}
	if !strings.Contains(m.View().Content, "passphrase") {
		t.Errorf("modal should mention passphrase;\n%s", m.View().Content)
	}

	// Dismiss modal.
	m2, _ = m.Update(modal.DismissMsg{})
	m = m2.(Model)
	if m.showModal {
		t.Fatal("modal should be hidden after DismissMsg")
	}
	if !strings.Contains(strings.ToLower(m.View().Content), "passphrase") {
		t.Errorf("passphrase prompt should be visible after dismiss;\n%s", m.View().Content)
	}

	// Now type the correct passphrase.
	for _, ch := range e2eCorrectPass {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	_, successCmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	success := runCmd(successCmd)
	if _, ok := success.(AuthSuccessMsg); !ok {
		t.Fatalf("expected AuthSuccessMsg after correct passphrase, got %T", success)
	}
}

// TestE2E_Auth_EscCancels verifies Esc emits AuthCancelMsg.
func TestE2E_Auth_EscCancels(t *testing.T) {
	m := newModel(t, e2eKeyPassphrase)
	_, cancelCmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	msg := runCmd(cancelCmd)
	if _, ok := msg.(AuthCancelMsg); !ok {
		t.Fatalf("expected AuthCancelMsg on Esc, got %T", msg)
	}
}

// TestE2E_Auth_InputMasked verifies that typed characters appear as asterisks.
func TestE2E_Auth_InputMasked(t *testing.T) {
	m := newModel(t, e2eKeyPassphrase)
	for _, ch := range "secret" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	view := m.View().Content
	if strings.Contains(view, "secret") {
		t.Errorf("typed passphrase should not appear in plain text;\n%s", view)
	}
	if !strings.Contains(view, "******") {
		t.Errorf("typed characters should appear as asterisks;\n%s", view)
	}
}
