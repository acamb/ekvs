package footer_test

import (
	"strings"
	"testing"

	"ekvs/internal/tui/footer"
	"ekvs/internal/tui/theme"
)

func newFooter(t *testing.T, themeName string) footer.Model {
	t.Helper()
	th, err := theme.NewTheme(themeName)
	if err != nil {
		t.Fatalf("NewTheme(%q): %v", themeName, err)
	}
	return footer.New(th)
}

// TestFooter_ViewContainsHints verifies that the rendered string contains the
// hints text passed to View.
func TestFooter_ViewContainsHints(t *testing.T) {
	tests := []struct {
		name  string
		hints string
	}{
		{name: "normal hints", hints: "↑/↓ navigate • Esc back"},
		{name: "empty hints", hints: ""},
		{name: "single word", hints: "quit"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newFooter(t, "adaptive")
			got := f.View(tc.hints)
			if tc.hints != "" && !strings.Contains(got, tc.hints) {
				t.Errorf("View(%q) = %q, does not contain hints", tc.hints, got)
			}
		})
	}
}

// TestFooter_ViewNoPanic verifies that View does not panic for either theme.
func TestFooter_ViewNoPanic(t *testing.T) {
	for _, name := range []string{"adaptive", "hacker"} {
		t.Run(name, func(t *testing.T) {
			f := newFooter(t, name)
			_ = f.View("↑/↓ navigate • Esc back")
			_ = f.View("")
		})
	}
}
