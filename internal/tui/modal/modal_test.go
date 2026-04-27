package modal_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/theme"
)

func newModal(t *testing.T, message string) modal.Model {
	t.Helper()
	th, err := theme.NewTheme("adaptive")
	if err != nil {
		t.Fatal(err)
	}
	return modal.New(th, message)
}

// TestModal_ViewContainsMessage verifies that View includes the error message.
func TestModal_ViewContainsMessage(t *testing.T) {
	m := newModal(t, "connection refused")
	got := m.View(80)
	if !strings.Contains(got, "connection refused") {
		t.Errorf("View(80) = %q\ndoes not contain the error message", got)
	}
}

// TestModal_ViewContainsTitle verifies that View includes the "Error" title.
func TestModal_ViewContainsTitle(t *testing.T) {
	m := newModal(t, "something went wrong")
	got := m.View(80)
	if !strings.Contains(got, "Error") {
		t.Errorf("View(80) does not contain 'Error' title:\n%s", got)
	}
}

// TestModal_ViewContainsDismissHint verifies that the dismiss hint is present.
func TestModal_ViewContainsDismissHint(t *testing.T) {
	m := newModal(t, "oops")
	got := m.View(80)
	if !strings.Contains(got, "Enter") {
		t.Errorf("View(80) does not contain dismiss hint 'Enter':\n%s", got)
	}
}

// TestModal_View_ZeroWidthNoPanic verifies that View(0) does not panic and
// uses a sensible fallback width.
func TestModal_View_ZeroWidthNoPanic(t *testing.T) {
	m := newModal(t, "err")
	got := m.View(0)
	if got == "" {
		t.Error("View(0) returned empty string")
	}
}

// TestModal_Update_EnterEmitsDismiss verifies that pressing Enter emits DismissMsg.
func TestModal_Update_EnterEmitsDismiss(t *testing.T) {
	m := newModal(t, "err")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Update(Enter) returned nil cmd, expected DismissMsg command")
	}
	msg := cmd()
	if _, ok := msg.(modal.DismissMsg); !ok {
		t.Errorf("cmd() = %T, want modal.DismissMsg", msg)
	}
}

// TestModal_Update_EscEmitsDismiss verifies that pressing Esc emits DismissMsg.
func TestModal_Update_EscEmitsDismiss(t *testing.T) {
	m := newModal(t, "err")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("Update(Esc) returned nil cmd, expected DismissMsg command")
	}
	msg := cmd()
	if _, ok := msg.(modal.DismissMsg); !ok {
		t.Errorf("cmd() = %T, want modal.DismissMsg", msg)
	}
}

// TestModal_Update_OtherKeyIgnored verifies that other keys do not dismiss the modal.
func TestModal_Update_OtherKeyIgnored(t *testing.T) {
	m := newModal(t, "err")
	_, cmd := m.Update(tea.KeyPressMsg{Code: rune('x'), Text: "x"})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(modal.DismissMsg); ok {
			t.Error("pressing 'x' should NOT emit DismissMsg")
		}
	}
}

// TestModal_BothThemes verifies that View does not panic on either theme.
func TestModal_BothThemes(t *testing.T) {
	for _, name := range []string{"adaptive", "hacker"} {
		th, err := theme.NewTheme(name)
		if err != nil {
			t.Fatalf("NewTheme(%q): %v", name, err)
		}
		m := modal.New(th, "test error message")
		got := m.View(80)
		if got == "" {
			t.Errorf("theme %q: View(80) returned empty string", name)
		}
	}
}
