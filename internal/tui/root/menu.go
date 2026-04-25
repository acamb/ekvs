package root

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/theme"
)

type menuItem struct {
	Label string
	ID    string
}

// defaultMenuItems returns a fresh copy of the main menu items on each call,
// preventing accidental mutation of a shared global slice.
func defaultMenuItems() []menuItem {
	return []menuItem{
		{ID: "projects", Label: "Projects"},
		{ID: "secrets", Label: "Secrets"},
		{ID: "settings", Label: "Settings"},
		{ID: "quit", Label: "Quit"},
	}
}

type mainModel struct {
	items  []menuItem
	cursor int
	theme  theme.Theme
}

func newMainModel(t theme.Theme) mainModel {
	return mainModel{items: defaultMenuItems(), theme: t}
}

func (m mainModel) Init() tea.Cmd { return nil }

// updateTyped returns the concrete type directly, avoiding silent type-assertion
// failures in the root model.
func (m mainModel) updateTyped(msg tea.Msg) (mainModel, tea.Cmd) {
	next, cmd := m.Update(msg)
	if mm, ok := next.(mainModel); ok {
		return mm, cmd
	}
	return m, cmd
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = moveCursor(m.cursor, -1, len(m.items))
		case "down", "j":
			m.cursor = moveCursor(m.cursor, +1, len(m.items))
		case "enter":
			if m.items[m.cursor].ID == "quit" {
				return m, tea.Quit
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m mainModel) View() tea.View {
	var sb strings.Builder
	sb.WriteString(m.theme.TitleStyle().Render("EKVS"))
	sb.WriteString("\n")
	for i, item := range m.items {
		if i == m.cursor {
			sb.WriteString(m.theme.SelectedMenuItemStyle().Render(fmt.Sprintf("> %s", item.Label)))
		} else {
			sb.WriteString(m.theme.MenuItemStyle().Render(fmt.Sprintf("  %s", item.Label)))
		}
		sb.WriteString("\n")
	}
	sb.WriteString(m.theme.StatusBarStyle().Render("↑/↓ navigate • Enter select • q quit"))
	return tea.NewView(sb.String())
}
