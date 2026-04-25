package theme_test

import (
	"testing"

	"ekvs/internal/tui/theme"
)

func TestNewTheme(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantType string
	}{
		{name: "adaptive ok", input: "adaptive", wantType: "AdaptiveTheme"},
		{name: "hacker ok", input: "hacker", wantType: "HackerTheme"},
		{name: "empty string returns error", input: "", wantErr: true},
		{name: "unknown name returns error", input: "neon", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := theme.NewTheme(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected non-nil theme")
			}
		})
	}
}

// smokeTheme verifies that all Theme interface methods are implemented
// and return non-nil values.
func smokeTheme(t *testing.T, th theme.Theme) {
	t.Helper()

	if th.PrimaryColor() == nil {
		t.Error("PrimaryColor is nil")
	}
	if th.SecondaryColor() == nil {
		t.Error("SecondaryColor is nil")
	}
	if th.BackgroundColor() == nil {
		t.Error("BackgroundColor is nil")
	}
	if th.ErrorColor() == nil {
		t.Error("ErrorColor is nil")
	}
	// Styles: just verify calls don't panic
	_ = th.TitleStyle()
	_ = th.MenuItemStyle()
	_ = th.SelectedMenuItemStyle()
	_ = th.StatusBarStyle()
	_ = th.ErrorStyle()
}

func TestAdaptiveThemeSmoke(t *testing.T) {
	th, err := theme.NewTheme("adaptive")
	if err != nil {
		t.Fatal(err)
	}
	smokeTheme(t, th)
}

func TestHackerThemeSmoke(t *testing.T) {
	th, err := theme.NewTheme("hacker")
	if err != nil {
		t.Fatal(err)
	}
	smokeTheme(t, th)
}
