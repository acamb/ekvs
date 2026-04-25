package root

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/auth"
	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/theme"
)

func newRootWithProfile(t *testing.T, identityFile string) Model {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	cfg := &tuiconfig.ConfigFile{
		Profiles: []tuiconfig.Profile{
			{Name: "test", ServerURL: "http://localhost:8080", IdentityFile: identityFile, Theme: "adaptive"},
		},
	}
	return New(cfg, th)
}

func runRootCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// TestRoot_AuthTrigger verifies that dispatching triggerAuthMsg transitions to screenAuth.
func TestRoot_AuthTrigger(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	if m.screen != screenMain {
		t.Fatalf("expected screenMain, got %v", m.screen)
	}

	next, _ := m.Update(triggerAuthMsg{returnTo: screenMain})
	rm := next.(Model)
	if rm.screen != screenAuth {
		t.Errorf("expected screenAuth after triggerAuthMsg, got %v", rm.screen)
	}
}

// TestRoot_AuthSuccess verifies that AuthSuccessMsg populates the session and
// returns to pendingScreen.
func TestRoot_AuthSuccess(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	m.pendingScreen = screenMain

	next, _ := m.Update(auth.AuthSuccessMsg{
		Signer:      nil,
		PublicKey:   nil,
		Fingerprint: "SHA256:test",
	})
	rm := next.(Model)
	if rm.screen != screenMain {
		t.Errorf("expected screenMain after AuthSuccessMsg, got %v", rm.screen)
	}
	if rm.session.Fingerprint != "SHA256:test" {
		t.Errorf("expected fingerprint SHA256:test, got %q", rm.session.Fingerprint)
	}
}

// TestRoot_AuthCancel verifies that AuthCancelMsg returns to screenMain without
// establishing a session.
func TestRoot_AuthCancel(t *testing.T) {
	m := newRootWithProfile(t, "../../../internal/ssh/testdata/ed25519")
	m.screen = screenAuth

	next, _ := m.Update(auth.AuthCancelMsg{})
	rm := next.(Model)
	if rm.screen != screenMain {
		t.Errorf("expected screenMain after AuthCancelMsg, got %v", rm.screen)
	}
	if rm.session.IsAuthenticated() {
		t.Error("session should not be authenticated after cancel")
	}
}
