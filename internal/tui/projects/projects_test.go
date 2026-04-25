package projects

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/theme"
)

// ── fake client ───────────────────────────────────────────────────────────────

type fakeClient struct {
	projects     []string
	listErr      error
	createErr    error
	deleteErr    error
	deleteCalled string
	createCalled string
}

func (f *fakeClient) ListProjects() ([]string, error) {
	return f.projects, f.listErr
}
func (f *fakeClient) CreateProject(name string) error {
	f.createCalled = name
	return f.createErr
}
func (f *fakeClient) DeleteProject(name string) error {
	f.deleteCalled = name
	return f.deleteErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestModel(t *testing.T, projects []string) Model {
	t.Helper()
	th, _ := theme.NewTheme("adaptive")
	fc := &fakeClient{projects: projects}
	return newWithClient(fc, th)
}

// keyMsg builds a tea.KeyPressMsg for a named or single-character key.
func keyMsg(key string) tea.KeyPressMsg {
	switch key {
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "left":
		return tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		return tea.KeyPressMsg{Code: tea.KeyRight}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	default:
		r := []rune(key)[0]
		return tea.KeyPressMsg{Code: r, Text: key}
	}
}

// sendKey simulates a key press on the model.
func sendKey(m Model, key string) (Model, tea.Cmd) {
	next, cmd := m.Update(keyMsg(key))
	mm, _ := next.(Model)
	return mm, cmd
}

// applyFetched delivers a FetchedMsg to the model (simulates server response).
func applyFetched(m Model, projects []string) Model {
	next, _ := m.Update(FetchedMsg{Projects: projects})
	mm, _ := next.(Model)
	return mm
}

// ── FetchedMsg ────────────────────────────────────────────────────────────────

func TestProjectsModel_FetchUpdates(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"alpha", "beta", "gamma"})

	if len(m.projects) != 3 {
		t.Errorf("want 3 projects, got %d", len(m.projects))
	}
	if m.loading {
		t.Error("loading should be false after FetchedMsg")
	}
	if m.err != nil {
		t.Errorf("err should be nil, got %v", m.err)
	}
}

// ── ErrMsg ────────────────────────────────────────────────────────────────────

func TestProjectsModel_ErrDisplayed(t *testing.T) {
	m := newTestModel(t, nil)
	next, _ := m.Update(ErrMsg{Err: errors.New("boom")})
	mm := next.(Model)

	if mm.err == nil {
		t.Fatal("expected err to be set")
	}
	if mm.loading {
		t.Error("loading should be false after ErrMsg")
	}
	view := mm.View().Content
	if !strings.Contains(view, "boom") {
		t.Errorf("view should contain error text, got:\n%s", view)
	}
}

// ── Cursor movement ───────────────────────────────────────────────────────────

func TestProjectsModel_CursorDown(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"a", "b", "c"})

	m, _ = sendKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("want cursor=1, got %d", m.cursor)
	}
}

func TestProjectsModel_CursorWrap(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"a", "b", "c"})

	// At 0, go up → should wrap to 2.
	m, _ = sendKey(m, "up")
	if m.cursor != 2 {
		t.Errorf("want cursor=2 after wrap-up, got %d", m.cursor)
	}
	// Then down three times → back to 2 wrapping through 0, 1, 2.
	for i := 0; i < 3; i++ {
		m, _ = sendKey(m, "down")
	}
	if m.cursor != 2 {
		t.Errorf("want cursor=2 after wrap-down, got %d", m.cursor)
	}
}

// ── Pagination ────────────────────────────────────────────────────────────────

func TestProjectsModel_Pagination(t *testing.T) {
	list := make([]string, 15)
	for i := range list {
		list[i] = "p"
	}

	m := newTestModel(t, nil)
	m = applyFetched(m, list)

	if m.page != 0 {
		t.Errorf("want page=0, got %d", m.page)
	}
	if len(m.pageProjects()) != 10 {
		t.Errorf("want 10 items on first page, got %d", len(m.pageProjects()))
	}

	m, _ = sendKey(m, "right")
	if m.page != 1 {
		t.Errorf("want page=1 after →, got %d", m.page)
	}
	if len(m.pageProjects()) != 5 {
		t.Errorf("want 5 items on second page, got %d", len(m.pageProjects()))
	}
}

func TestProjectsModel_PaginationBoundary(t *testing.T) {
	list := make([]string, 15)
	for i := range list {
		list[i] = "p"
	}

	m := newTestModel(t, nil)
	m = applyFetched(m, list)

	// left on page 0 → stays at 0
	m, _ = sendKey(m, "left")
	if m.page != 0 {
		t.Errorf("want page=0 on left boundary, got %d", m.page)
	}

	// Navigate to last page, then right → stays
	m, _ = sendKey(m, "right") // page 1 (last)
	m, _ = sendKey(m, "right") // should stay at 1
	if m.page != 1 {
		t.Errorf("want page=1 on right boundary, got %d", m.page)
	}
}

// ── Create mode ───────────────────────────────────────────────────────────────

func TestProjectsModel_CreateMode(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"existing"})

	m, _ = sendKey(m, "n")
	if m.mode != modeCreate {
		t.Fatalf("want modeCreate, got %v", m.mode)
	}

	// Type characters.
	for _, ch := range "hello" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	if m.input != "hello" {
		t.Errorf("want input=hello, got %q", m.input)
	}

	// Backspace removes last rune.
	m, _ = sendKey(m, "backspace")
	if m.input != "hell" {
		t.Errorf("want input=hell after backspace, got %q", m.input)
	}
}

func TestProjectsModel_CreateValidation(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{})

	m, _ = sendKey(m, "n") // enter create mode

	// Submit empty name.
	m, _ = sendKey(m, "enter")
	if m.mode != modeCreate {
		t.Error("should stay in modeCreate after empty submit")
	}
	if m.err == nil {
		t.Error("expected validation error for empty name")
	}
}

func TestProjectsModel_CreateSubmit(t *testing.T) {
	fc := &fakeClient{projects: []string{"new-proj"}}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)
	m = applyFetched(m, []string{})

	m, _ = sendKey(m, "n")
	for _, ch := range "new-proj" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}

	m2, cmd := sendKey(m, "enter")
	if m2.loading != true {
		t.Error("want loading=true after create submit")
	}
	if cmd == nil {
		t.Fatal("want non-nil cmd after create submit")
	}
	// Run the command, get FetchedMsg.
	result := cmd()
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("want FetchedMsg, got %T: %v", result, result)
	}
	if fc.createCalled != "new-proj" {
		t.Errorf("want CreateProject called with new-proj, got %q", fc.createCalled)
	}
}

func TestProjectsModel_CreateCancel(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{})

	m, _ = sendKey(m, "n")
	m, _ = sendKey(m, "esc")
	if m.mode != modeList {
		t.Errorf("want modeList after esc, got %v", m.mode)
	}
}

// ── Delete mode ───────────────────────────────────────────────────────────────

func TestProjectsModel_DeleteConfirm(t *testing.T) {
	fc := &fakeClient{projects: []string{}}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)
	m = applyFetched(m, []string{"to-delete"})

	m, _ = sendKey(m, "d")
	if m.mode != modeDelete {
		t.Fatalf("want modeDelete, got %v", m.mode)
	}

	m2, cmd := sendKey(m, "y")
	if m2.loading != true {
		t.Error("want loading=true after delete confirm")
	}
	if cmd == nil {
		t.Fatal("want non-nil cmd after delete confirm")
	}
	result := cmd()
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("want FetchedMsg, got %T: %v", result, result)
	}
	if fc.deleteCalled != "to-delete" {
		t.Errorf("want DeleteProject called with to-delete, got %q", fc.deleteCalled)
	}
}

func TestProjectsModel_DeleteCancel(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"keep-me"})

	m, _ = sendKey(m, "d")

	// Cancel with n.
	m, _ = sendKey(m, "n")
	if m.mode != modeList {
		t.Errorf("want modeList after n, got %v", m.mode)
	}

	// Also test esc.
	m, _ = sendKey(m, "d")
	m, _ = sendKey(m, "esc")
	if m.mode != modeList {
		t.Errorf("want modeList after esc, got %v", m.mode)
	}
}

// ── Loading state ─────────────────────────────────────────────────────────────

func TestProjectsModel_LoadingState(t *testing.T) {
	m := newTestModel(t, nil)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init should return a fetch cmd")
	}
	// loading is set inside Init — verify by running command and checking model response.
	result := cmd()
	if _, ok := result.(FetchedMsg); !ok {
		t.Errorf("want FetchedMsg from Init cmd, got %T", result)
	}

	// After FetchedMsg, loading must be false.
	m = applyFetched(m, []string{})
	if m.loading {
		t.Error("loading should be false after FetchedMsg")
	}
}

// ── BackMsg on Esc ────────────────────────────────────────────────────────────

func TestProjectsModel_BackOnEsc(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{})

	_, cmd := sendKey(m, "esc")
	if cmd == nil {
		t.Fatal("want non-nil cmd on esc")
	}
	msg := cmd()
	if _, ok := msg.(BackMsg); !ok {
		t.Errorf("want BackMsg, got %T", msg)
	}
}

// ── ErrMsg methods ────────────────────────────────────────────────────────────

func TestErrMsg_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	e := ErrMsg{Err: inner}
	if e.Error() != "inner error" {
		t.Errorf("got %q, want %q", e.Error(), "inner error")
	}
	if !errors.Is(e, inner) {
		t.Error("errors.Is should find the inner error via Unwrap")
	}

	// nil Err should not panic.
	empty := ErrMsg{}
	if empty.Error() != "" {
		t.Errorf("empty ErrMsg.Error() should return empty string, got %q", empty.Error())
	}
}

// ── j/k aliases ───────────────────────────────────────────────────────────────

func TestProjectsModel_JKAliases(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"a", "b", "c"})

	m, _ = sendKey(m, "j") // down
	if m.cursor != 1 {
		t.Errorf("j: want cursor=1, got %d", m.cursor)
	}
	m, _ = sendKey(m, "k") // up
	if m.cursor != 0 {
		t.Errorf("k: want cursor=0, got %d", m.cursor)
	}
}

// ── UpdateTyped ───────────────────────────────────────────────────────────────

func TestProjectsModel_UpdateTyped(t *testing.T) {
	m := newTestModel(t, nil)
	got, _ := m.UpdateTyped(FetchedMsg{Projects: []string{"x"}})
	if len(got.projects) != 1 {
		t.Errorf("UpdateTyped: want 1 project, got %d", len(got.projects))
	}
}

// ── Path separator validation ─────────────────────────────────────────────────

func TestProjectsModel_CreatePathSeparator(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{})
	m, _ = sendKey(m, "n")

	for _, ch := range "bad/name" {
		next, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = next.(Model)
	}
	m, _ = sendKey(m, "enter")
	if m.err == nil {
		t.Error("expected validation error for name with path separator")
	}
	if m.mode != modeCreate {
		t.Error("should stay in modeCreate after path separator validation error")
	}
}

// ── cmd error paths ───────────────────────────────────────────────────────────

func TestProjectsModel_FetchCmdError(t *testing.T) {
	fc := &fakeClient{listErr: errors.New("network fail")}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)

	cmd := m.fetchCmd()
	msg := cmd()
	if errMsg, ok := msg.(ErrMsg); !ok {
		t.Errorf("want ErrMsg, got %T", msg)
	} else if errMsg.Err == nil {
		t.Error("ErrMsg.Err should be set")
	}
}

func TestProjectsModel_CreateCmdError(t *testing.T) {
	fc := &fakeClient{createErr: errors.New("server error")}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)

	cmd := m.createCmd("proj")
	msg := cmd()
	if _, ok := msg.(ErrMsg); !ok {
		t.Errorf("want ErrMsg on create failure, got %T", msg)
	}
}

func TestProjectsModel_CreateCmdListErrorAfterCreate(t *testing.T) {
	fc := &fakeClient{listErr: errors.New("list fail")}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)

	cmd := m.createCmd("proj")
	msg := cmd()
	if _, ok := msg.(ErrMsg); !ok {
		t.Errorf("want ErrMsg when list fails after create, got %T", msg)
	}
}

func TestProjectsModel_DeleteCmdError(t *testing.T) {
	fc := &fakeClient{deleteErr: errors.New("not found")}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)

	cmd := m.deleteCmd("proj")
	msg := cmd()
	if _, ok := msg.(ErrMsg); !ok {
		t.Errorf("want ErrMsg on delete failure, got %T", msg)
	}
}

func TestProjectsModel_DeleteCmdListErrorAfterDelete(t *testing.T) {
	fc := &fakeClient{listErr: errors.New("list fail")}
	th, _ := theme.NewTheme("adaptive")
	m := newWithClient(fc, th)

	cmd := m.deleteCmd("proj")
	msg := cmd()
	if _, ok := msg.(ErrMsg); !ok {
		t.Errorf("want ErrMsg when list fails after delete, got %T", msg)
	}
}

// ── View modes ────────────────────────────────────────────────────────────────

func TestProjectsModel_ViewCreateMode(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"existing"})
	m, _ = sendKey(m, "n")

	view := m.View().Content
	if !strings.Contains(view, "New project name:") {
		t.Errorf("create mode view should show input prompt, got:\n%s", view)
	}
}

func TestProjectsModel_ViewDeleteMode(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"target"})
	m, _ = sendKey(m, "d")

	view := m.View().Content
	if !strings.Contains(view, "Delete") || !strings.Contains(view, "target") {
		t.Errorf("delete mode view should show confirmation, got:\n%s", view)
	}
}

func TestProjectsModel_ViewLoadingState(t *testing.T) {
	m := newTestModel(t, nil)
	m.loading = true

	view := m.View().Content
	if !strings.Contains(view, "Loading") {
		t.Errorf("loading view should contain 'Loading', got:\n%s", view)
	}
}

func TestProjectsModel_ViewEmptyList(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{})

	view := m.View().Content
	if !strings.Contains(view, "no projects") {
		t.Errorf("empty list view should say 'no projects', got:\n%s", view)
	}
}

// ── selectedName edge case ────────────────────────────────────────────────────

func TestProjectsModel_SelectedNameEmpty(t *testing.T) {
	m := newTestModel(t, nil)
	// No projects loaded → selectedName should return "".
	if got := m.selectedName(); got != "" {
		t.Errorf("want empty selectedName, got %q", got)
	}
}

// ── delete with empty list (no-op) ────────────────────────────────────────────

func TestProjectsModel_DeleteOnEmptyList(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{})

	// "d" on empty list should not enter delete mode.
	m, _ = sendKey(m, "d")
	if m.mode != modeList {
		t.Errorf("want modeList when pressing d on empty list, got %v", m.mode)
	}
}

// ── enter opens secrets ───────────────────────────────────────────────────────

func TestProjectsModel_EnterOpensSecrets(t *testing.T) {
	m := newTestModel(t, nil)
	m = applyFetched(m, []string{"myproject"})

	_, cmd := sendKey(m, "enter")
	if cmd == nil {
		t.Fatal("expected a command, got nil")
	}
	msg := cmd()
	osm, ok := msg.(OpenSecretsMsg)
	if !ok {
		t.Fatalf("want OpenSecretsMsg, got %T", msg)
	}
	if osm.Project != "myproject" {
		t.Errorf("want Project=%q, got %q", "myproject", osm.Project)
	}
}
