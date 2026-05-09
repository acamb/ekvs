package projects

// projects_e2e_test.go — end-to-end user-journey tests for the Projects screen.
//
// These tests drive the model through realistic multi-step interaction sequences
// and assert on the rendered View content, complementing the unit tests in
// projects_test.go.

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/theme"
)

func e2eTheme(t *testing.T) theme.Theme {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	return th
}

// TestE2E_Projects_LoadRendersProjectList verifies the full init→fetch→list flow.
func TestE2E_Projects_LoadRendersProjectList(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"alpha", "beta", "gamma"})

	view := m.View().Content
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(view, name) {
			t.Errorf("view should contain project %q;\n%s", name, view)
		}
	}
	if !strings.Contains(view, "> alpha") {
		t.Errorf("first item should be highlighted (> alpha);\n%s", view)
	}
}

// TestE2E_Projects_NavigateChangesHighlight verifies cursor movement changes the view.
func TestE2E_Projects_NavigateChangesHighlight(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"first", "second", "third"})

	if !strings.Contains(m.View().Content, "> first") {
		t.Fatalf("expected '> first' initially;\n%s", m.View().Content)
	}
	m, _ = sendKey(m, "down")
	if !strings.Contains(m.View().Content, "> second") {
		t.Errorf("expected '> second' after ↓;\n%s", m.View().Content)
	}
	m, _ = sendKey(m, "up")
	if !strings.Contains(m.View().Content, "> first") {
		t.Errorf("expected '> first' after ↑;\n%s", m.View().Content)
	}
}

// TestE2E_Projects_CreateProjectFlow tests the complete create user journey.
func TestE2E_Projects_CreateProjectFlow(t *testing.T) {
	fc := &fakeClient{projects: []string{"newproj"}}
	m := newWithClient(fc, e2eTheme(t))
	m = applyFetched(m, []string{})

	m, _ = sendKey(m, "n")
	if m.mode != modeCreate {
		t.Fatalf("expected modeCreate, got %v", m.mode)
	}
	for _, ch := range "newproj" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	if !strings.Contains(m.View().Content, "newproj") {
		t.Errorf("typed name should appear in create prompt;\n%s", m.View().Content)
	}

	m2, cmd := sendKey(m, "enter")
	if !m2.loading {
		t.Error("expected loading=true after create submit")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd after create submit")
	}
	result := cmd()
	if fc.createCalled != "newproj" {
		t.Errorf("CreateProject not called with 'newproj'; got %q", fc.createCalled)
	}
	fetched, ok := result.(FetchedMsg)
	if !ok {
		t.Fatalf("expected FetchedMsg, got %T", result)
	}
	m2 = applyFetched(m2, fetched.Projects)
	if !strings.Contains(m2.View().Content, "newproj") {
		t.Errorf("'newproj' should appear in list after create;\n%s", m2.View().Content)
	}
}

// TestE2E_Projects_DeleteProjectFlow tests the full delete journey.
func TestE2E_Projects_DeleteProjectFlow(t *testing.T) {
	fc := &fakeClient{projects: []string{"keep"}}
	m := newWithClient(fc, e2eTheme(t))
	m = applyFetched(m, []string{"to-delete", "keep"})

	m, _ = sendKey(m, "d")
	if m.mode != modeDelete {
		t.Fatalf("expected modeDelete, got %v", m.mode)
	}
	if !strings.Contains(m.View().Content, "to-delete") {
		t.Errorf("delete prompt should mention the selected project;\n%s", m.View().Content)
	}

	m2, cmd := sendKey(m, "y")
	if !m2.loading {
		t.Error("expected loading=true after 'y'")
	}
	result := cmd()
	if fc.deleteCalled != "to-delete" {
		t.Errorf("DeleteProject not called with 'to-delete'; got %q", fc.deleteCalled)
	}
	fetched, ok := result.(FetchedMsg)
	if !ok {
		t.Fatalf("expected FetchedMsg, got %T", result)
	}
	m2 = applyFetched(m2, fetched.Projects)
	if strings.Contains(m2.View().Content, "to-delete") {
		t.Errorf("'to-delete' should not appear after delete;\n%s", m2.View().Content)
	}
	if !strings.Contains(m2.View().Content, "keep") {
		t.Errorf("'keep' should still appear;\n%s", m2.View().Content)
	}
}

// TestE2E_Projects_ErrorModalDismissFlow tests the error→modal→dismiss cycle.
func TestE2E_Projects_ErrorModalDismissFlow(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"alpha"})

	next, _ := m.Update(ErrMsg{Err: errors.New("network failure")})
	m = next.(Model)
	if m.mode != modeError {
		t.Fatalf("expected modeError, got %v", m.mode)
	}
	view := m.View().Content
	if !strings.Contains(view, "network failure") {
		t.Errorf("view should contain error message;\n%s", view)
	}
	if !strings.Contains(view, "Error") {
		t.Errorf("view should contain modal title 'Error';\n%s", view)
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
	if strings.Contains(m2.View().Content, "network failure") {
		t.Errorf("error message should be gone after dismiss;\n%s", m2.View().Content)
	}
}

// TestE2E_Projects_FooterHintsInAllModes verifies footer hints across modes.
func TestE2E_Projects_FooterHintsInAllModes(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"p1"})

	view := m.View().Content
	if !strings.Contains(view, "n") || !strings.Contains(view, "d") {
		t.Errorf("list mode footer should contain 'n' and 'd';\n%s", view)
	}

	m, _ = sendKey(m, "n")
	view = m.View().Content
	if !strings.Contains(view, "confirm") || !strings.Contains(view, "cancel") {
		t.Errorf("create mode footer should contain hints;\n%s", view)
	}

	m2 := newTestModel(t, nil)
	m2 = applyFetched(m2, []string{"p1"})
	m2, _ = sendKey(m2, "d")
	view2 := m2.View().Content
	if !strings.Contains(view2, "y") || !strings.Contains(view2, "n") {
		t.Errorf("delete mode footer should contain 'y'/'n';\n%s", view2)
	}
}

// TestE2E_Projects_SpinnerVisibleDuringLoad verifies the spinner during loading.
func TestE2E_Projects_SpinnerVisibleDuringLoad(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true
	view := m.View().Content
	if !strings.Contains(view, "Loading") {
		t.Errorf("loading view should contain 'Loading';\n%s", view)
	}
}
