package secrets

// secrets_e2e_test.go — end-to-end user-journey tests for the Secrets screen.

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/client"
	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/theme"
)

func e2eSecretsTheme(t *testing.T) theme.Theme {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	return th
}

// TestE2E_Secrets_LoadRendersTable verifies the fetch→table flow.
func TestE2E_Secrets_LoadRendersTable(t *testing.T) {
	sess := testSession(t)
	blob1 := encryptBlob(t, sess, "secretval1")
	blob2 := encryptBlob(t, sess, "secretval2")
	entries := []client.SecretEntry{
		{Key: "API_KEY", Value: blob1},
		{Key: "DB_PASS", Value: blob2},
	}
	m := newTestModel(t, entries)
	m = applyFetched(m, entries)

	view := m.View().Content
	for _, want := range []string{"KEY", "VALUE", "API_KEY", "DB_PASS"} {
		if !strings.Contains(view, want) {
			t.Errorf("view should contain %q;\n%s", want, view)
		}
	}
}

// TestE2E_Secrets_AddSecretFlow tests the full add journey with view assertions.
func TestE2E_Secrets_AddSecretFlow(t *testing.T) {
	sess := testSession(t)
	fc := &fakeClient{}
	m := newWithClient("proj", fc, sess, e2eSecretsTheme(t))
	m = applyFetched(m, []client.SecretEntry{})

	m, _ = sendKey(m, "n")
	if m.mode != modeAdd {
		t.Fatalf("expected modeAdd after 'n', got %v", m.mode)
	}

	for _, ch := range "MYKEY" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	if m.inputKey != "MYKEY" {
		t.Errorf("inputKey = %q, want 'MYKEY'", m.inputKey)
	}

	// Advance to value field.
	m, _ = sendKey(m, "enter")
	if m.activeField != 1 {
		t.Fatalf("expected activeField=1, got %d", m.activeField)
	}

	for _, ch := range "mysecret" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}

	// Submit.
	m2, cmd := sendKey(m, "enter")
	if !m2.loading {
		t.Error("expected loading=true after submit")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd after submit")
	}
	result := cmd()
	if fc.setCalled[0] != "MYKEY" {
		t.Errorf("SetSecret key = %q, want 'MYKEY'", fc.setCalled[0])
	}
	if fc.setCalled[1] == "" {
		t.Error("SetSecret encrypted value should not be empty")
	}
	if _, ok := result.(FetchedMsg); !ok {
		t.Fatalf("expected FetchedMsg, got %T", result)
	}
}

// TestE2E_Secrets_DeleteSecretFlow tests the full delete journey.
func TestE2E_Secrets_DeleteSecretFlow(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "val")
	fc := &fakeClient{secrets: []client.SecretEntry{{Key: "OLD_KEY", Value: blob}}}
	m := newWithClient("proj", fc, sess, e2eSecretsTheme(t))
	m = applyFetched(m, []client.SecretEntry{{Key: "OLD_KEY", Value: blob}})

	m, _ = sendKey(m, "d")
	if m.mode != modeDelete {
		t.Fatalf("expected modeDelete, got %v", m.mode)
	}
	if !strings.Contains(m.View().Content, "OLD_KEY") {
		t.Errorf("delete prompt should mention key;\n%s", m.View().Content)
	}

	m2, cmd := sendKey(m, "y")
	if !m2.loading {
		t.Error("expected loading=true after 'y'")
	}
	result := cmd()
	if fc.deleteCalled != "OLD_KEY" {
		t.Errorf("DeleteSecret = %q, want 'OLD_KEY'", fc.deleteCalled)
	}
	if _, ok := result.(FetchedMsg); !ok {
		t.Fatalf("expected FetchedMsg, got %T", result)
	}
}

// TestE2E_Secrets_SearchAndClearCycle tests the / → filter → Esc → Esc clear cycle.
func TestE2E_Secrets_SearchAndClearCycle(t *testing.T) {
	sess := testSession(t)
	blobA := encryptBlob(t, sess, "va")
	blobB := encryptBlob(t, sess, "vb")
	entries := []client.SecretEntry{
		{Key: "APPLE_KEY", Value: blobA},
		{Key: "BANANA_KEY", Value: blobB},
	}
	m := newTestModel(t, entries)
	m = applyFetched(m, entries)

	m, _ = sendKey(m, "/")
	if m.mode != modeSearch {
		t.Fatalf("expected modeSearch, got %v", m.mode)
	}

	for _, ch := range "apple" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	if m.searchQuery != "apple" {
		t.Errorf("searchQuery = %q, want 'apple'", m.searchQuery)
	}
	view := m.View().Content
	if !strings.Contains(view, "APPLE_KEY") {
		t.Errorf("filtered view should contain APPLE_KEY;\n%s", view)
	}
	if strings.Contains(view, "BANANA_KEY") {
		t.Errorf("filtered view should NOT contain BANANA_KEY;\n%s", view)
	}

	// Esc → modeList, filter kept.
	m, _ = sendKey(m, "esc")
	if m.mode != modeList {
		t.Fatalf("expected modeList after Esc, got %v", m.mode)
	}
	if m.searchQuery == "" {
		t.Error("filter should be active after first Esc")
	}

	// Second Esc → clears filter.
	m, _ = sendKey(m, "esc")
	if m.searchQuery != "" {
		t.Errorf("searchQuery should be cleared after second Esc, got %q", m.searchQuery)
	}
	view2 := m.View().Content
	for _, want := range []string{"APPLE_KEY", "BANANA_KEY"} {
		if !strings.Contains(view2, want) {
			t.Errorf("%q should reappear after clear;\n%s", want, view2)
		}
	}
}

// TestE2E_Secrets_ErrorModalFlow tests the error→modal→dismiss cycle.
func TestE2E_Secrets_ErrorModalFlow(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	entries := []client.SecretEntry{{Key: "K", Value: blob}}
	m := newTestModel(t, entries)
	m = applyFetched(m, entries)

	next, _ := m.Update(ErrMsg{Err: errors.New("server down")})
	m = next.(Model)
	if m.mode != modeError {
		t.Fatalf("expected modeError, got %v", m.mode)
	}
	view := m.View().Content
	if !strings.Contains(view, "server down") {
		t.Errorf("view should contain error text;\n%s", view)
	}
	if !strings.Contains(view, "Error") {
		t.Errorf("view should contain modal title;\n%s", view)
	}

	m2, dismissCmd := m.UpdateTyped(tea.KeyPressMsg{Code: tea.KeyEnter})
	if dismissCmd != nil {
		if dmsg, ok := dismissCmd().(modal.DismissMsg); ok {
			m2, _ = m2.UpdateTyped(dmsg)
		}
	}
	if m2.mode != modeList {
		t.Errorf("expected modeList after dismiss, got %v", m2.mode)
	}
	if strings.Contains(m2.View().Content, "server down") {
		t.Errorf("error text should be gone after dismiss;\n%s", m2.View().Content)
	}
}

// TestE2E_Secrets_FooterHintsListMode verifies list-mode footer hints.
func TestE2E_Secrets_FooterHintsListMode(t *testing.T) {
	sess := testSession(t)
	blob := encryptBlob(t, sess, "v")
	entries := []client.SecretEntry{{Key: "K", Value: blob}}
	m := newTestModel(t, entries)
	m = applyFetched(m, entries)

	view := m.View().Content
	for _, want := range []string{"n", "d", "/"} {
		if !strings.Contains(view, want) {
			t.Errorf("list footer missing %q;\n%s", want, view)
		}
	}
}
