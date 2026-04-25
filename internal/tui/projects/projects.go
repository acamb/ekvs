package projects

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/client"
	"ekvs/internal/tui/theme"
)

const pageSize = 10

// mode represents the current interaction mode of the Projects screen.
type mode int

const (
	modeList   mode = iota
	modeCreate      // inline text input at the bottom
	modeDelete      // inline y/n confirmation at the bottom
)

// apiClient is the subset of client.Client used by this model.
// Defined as an interface to allow test fakes.
type apiClient interface {
	ListProjects() ([]string, error)
	CreateProject(name string) error
	DeleteProject(name string) error
}

// Model is the bubbletea model for the Projects screen.
type Model struct {
	client   apiClient
	theme    theme.Theme
	projects []string
	cursor   int // index within the current page
	page     int
	mode     mode
	input    string // accumulated text in modeCreate
	err      error
	loading  bool
}

// New creates a Model ready to be initialised.
func New(c *client.Client, t theme.Theme) Model {
	return Model{client: c, theme: t}
}

// newWithClient creates a Model using any apiClient implementation (used in tests).
func newWithClient(c apiClient, t theme.Theme) Model {
	return Model{client: c, theme: t}
}

// ── commands ─────────────────────────────────────────────────────────────────

func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.client.ListProjects()
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("fetch projects: %w", err)}
		}
		return FetchedMsg{Projects: projects}
	}
}

func (m Model) createCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.CreateProject(name); err != nil {
			return ErrMsg{Err: fmt.Errorf("create project: %w", err)}
		}
		// Re-fetch after successful create.
		projects, err := m.client.ListProjects()
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("fetch projects: %w", err)}
		}
		return FetchedMsg{Projects: projects}
	}
}

func (m Model) deleteCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteProject(name); err != nil {
			return ErrMsg{Err: fmt.Errorf("delete project: %w", err)}
		}
		// Re-fetch after successful delete.
		projects, err := m.client.ListProjects()
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("fetch projects: %w", err)}
		}
		return FetchedMsg{Projects: projects}
	}
}

// ── pagination helpers ────────────────────────────────────────────────────────

func (m Model) totalPages() int {
	if len(m.projects) == 0 {
		return 1
	}
	return (len(m.projects) + pageSize - 1) / pageSize
}

// pageProjects returns the slice of project names for the current page.
func (m Model) pageProjects() []string {
	start := m.page * pageSize
	if start >= len(m.projects) {
		return []string{}
	}
	end := start + pageSize
	if end > len(m.projects) {
		end = len(m.projects)
	}
	return m.projects[start:end]
}

// selectedName returns the name of the currently highlighted project,
// or "" when the list is empty.
func (m Model) selectedName() string {
	items := m.pageProjects()
	if len(items) == 0 || m.cursor >= len(items) {
		return ""
	}
	return items[m.cursor]
}

// ── Init / Update / View ──────────────────────────────────────────────────────

// Init implements tea.Model. Emits a fetch command immediately.
func (m Model) Init() tea.Cmd {
	m.loading = true
	return m.fetchCmd()
}

// UpdateTyped returns the concrete Model type for use by the root model.
func (m Model) UpdateTyped(msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	if mm, ok := next.(Model); ok {
		return mm, cmd
	}
	return m, cmd
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── server responses ────────────────────────────────────────────────────
	case FetchedMsg:
		m.projects = msg.Projects
		m.loading = false
		m.err = nil
		// Reset cursor when list changes.
		if m.cursor >= len(m.pageProjects()) {
			m.cursor = 0
		}
		m.mode = modeList
		return m, nil

	case ErrMsg:
		m.loading = false
		m.err = msg.Err
		m.mode = modeList
		return m, nil

	// ── keyboard ────────────────────────────────────────────────────────────
	case tea.KeyPressMsg:
		// Clear previous error on any key.
		m.err = nil

		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeCreate:
			return m.updateCreate(msg)
		case modeDelete:
			return m.updateDelete(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.pageProjects()
	switch msg.String() {
	case "up", "k":
		if len(items) > 0 {
			m.cursor = (m.cursor - 1 + len(items)) % len(items)
		}
	case "down", "j":
		if len(items) > 0 {
			m.cursor = (m.cursor + 1) % len(items)
		}
	case "right":
		if m.page < m.totalPages()-1 {
			m.page++
			m.cursor = 0
		}
	case "left":
		if m.page > 0 {
			m.page--
			m.cursor = 0
		}
	case "n":
		m.mode = modeCreate
		m.input = ""
	case "enter":
		name := m.selectedName()
		if name != "" {
			return m, func() tea.Msg { return OpenSecretsMsg{Project: name} }
		}
	case "d":
		if len(items) > 0 {
			m.mode = modeDelete
		}
	case "esc", "backspace":
		return m, func() tea.Msg { return BackMsg{} }
	}
	return m, nil
}

func (m Model) updateCreate(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.input = ""
	case "backspace":
		if len(m.input) > 0 {
			r := []rune(m.input)
			m.input = string(r[:len(r)-1])
		}
	case "enter":
		name := strings.TrimSpace(m.input)
		if name == "" {
			m.err = fmt.Errorf("name cannot be empty")
			return m, nil
		}
		if strings.ContainsAny(name, "/\\") {
			m.err = fmt.Errorf("name must not contain path separators")
			return m, nil
		}
		m.loading = true
		m.input = ""
		return m, m.createCmd(name)
	default:
		if text := msg.Key().Text; text != "" {
			m.input += text
		}
	}
	return m, nil
}

func (m Model) updateDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		name := m.selectedName()
		if name == "" {
			m.mode = modeList
			return m, nil
		}
		m.loading = true
		m.mode = modeList
		return m, m.deleteCmd(name)
	case "n", "esc":
		m.mode = modeList
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var sb strings.Builder

	sb.WriteString(m.theme.TitleStyle().Render("Projects"))
	sb.WriteString("\n")

	if m.loading {
		sb.WriteString(m.theme.MenuItemStyle().Render("  Loading…"))
		sb.WriteString("\n")
		return tea.NewView(sb.String())
	}

	items := m.pageProjects()
	if len(items) == 0 {
		sb.WriteString(m.theme.MenuItemStyle().Render("  (no projects)"))
		sb.WriteString("\n")
	} else {
		for i, name := range items {
			if i == m.cursor {
				sb.WriteString(m.theme.SelectedMenuItemStyle().Render(fmt.Sprintf("> %s", name)))
			} else {
				sb.WriteString(m.theme.MenuItemStyle().Render(fmt.Sprintf("  %s", name)))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	// Inline interaction rows.
	switch m.mode {
	case modeCreate:
		sb.WriteString(m.theme.MenuItemStyle().Render(fmt.Sprintf("  New project name: %s█", m.input)))
		sb.WriteString("\n")
	case modeDelete:
		name := m.selectedName()
		sb.WriteString(m.theme.ErrorStyle().Render(fmt.Sprintf("  Delete %q? [y/n]", name)))
		sb.WriteString("\n")
	}

	// Error line.
	if m.err != nil {
		sb.WriteString(m.theme.ErrorStyle().Render(fmt.Sprintf("  Error: %s", m.err.Error())))
		sb.WriteString("\n")
	}

	// Status bar.
	pageInfo := fmt.Sprintf("Page %d/%d", m.page+1, m.totalPages())
	var hints string
	switch m.mode {
	case modeList:
		hints = "↑/↓ navigate • ←/→ page • n new • d delete • Esc back"
	case modeCreate:
		hints = "Enter confirm • Esc cancel"
	case modeDelete:
		hints = "y confirm • n/Esc cancel"
	}
	sb.WriteString(m.theme.StatusBarStyle().Render(fmt.Sprintf("%s  %s", pageInfo, hints)))

	return tea.NewView(sb.String())
}
