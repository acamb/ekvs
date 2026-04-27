package spinner_test

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/spinner"
	"ekvs/internal/tui/theme"
)

func newSpinner(t *testing.T) spinner.Model {
	t.Helper()
	th, err := theme.NewTheme("adaptive")
	if err != nil {
		t.Fatal(err)
	}
	return spinner.New(th)
}

// TestSpinner_Init verifies that Init returns a non-nil command.
func TestSpinner_Init(t *testing.T) {
	m := newSpinner(t)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd, expected a tick command")
	}
}

// TestSpinner_TickAdvancesFrame verifies that each TickMsg advances the frame
// index and wraps around after the last frame.
func TestSpinner_TickAdvancesFrame(t *testing.T) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	m := newSpinner(t)

	for i := 0; i < len(frames)*2; i++ {
		want := frames[i%len(frames)]
		got := m.View()
		if !strings.Contains(got, want) {
			t.Errorf("tick %d: View() = %q, want frame %q", i, got, want)
		}
		m, _ = m.Update(spinner.TickMsg{})
	}
}

// TestSpinner_TickReturnsNextTickCmd verifies that each TickMsg produces
// another non-nil command (the next tick).
func TestSpinner_TickReturnsNextTickCmd(t *testing.T) {
	m := newSpinner(t)
	_, cmd := m.Update(spinner.TickMsg{})
	if cmd == nil {
		t.Fatal("Update(TickMsg) returned nil cmd, expected next tick command")
	}
}

// TestSpinner_NonTickMsgIgnored verifies that unknown messages do not change
// the frame and return a nil command.
func TestSpinner_NonTickMsgIgnored(t *testing.T) {
	m := newSpinner(t)
	before := m.View()
	m2, cmd := m.Update(tea.KeyPressMsg{})
	if cmd != nil {
		t.Error("non-TickMsg should return nil cmd")
	}
	if m2.View() != before {
		t.Error("non-TickMsg should not change the frame")
	}
}

// TestSpinner_ViewNonEmpty verifies that View always returns a non-empty string.
func TestSpinner_ViewNonEmpty(t *testing.T) {
	m := newSpinner(t)
	if m.View() == "" {
		t.Error("View() returned empty string")
	}
}

// TestSpinner_BothThemes verifies that the spinner works without panic on both
// supported themes.
func TestSpinner_BothThemes(t *testing.T) {
	for _, name := range []string{"adaptive", "hacker"} {
		th, err := theme.NewTheme(name)
		if err != nil {
			t.Fatalf("NewTheme(%q): %v", name, err)
		}
		m := spinner.New(th)
		if m.View() == "" {
			t.Errorf("theme %q: View() returned empty string", name)
		}
	}
}
