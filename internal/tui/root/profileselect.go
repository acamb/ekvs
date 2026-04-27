package root

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/footer"
	"ekvs/internal/tui/theme"
)

// profileChosenMsg is sent by profileSelectModel when the user confirms a profile.
type profileChosenMsg struct{ profile tuiconfig.Profile }

type profileSelectModel struct {
	profiles []tuiconfig.Profile
	cursor   int
	theme    theme.Theme
	footer   footer.Model
}

func newProfileSelectModel(profiles []tuiconfig.Profile, t theme.Theme) profileSelectModel {
	return profileSelectModel{profiles: profiles, theme: t, footer: footer.New(t)}
}

func (m profileSelectModel) Init() tea.Cmd { return nil }

// updateTyped returns the concrete type directly, avoiding silent type-assertion
// failures in the root model.
func (m profileSelectModel) updateTyped(msg tea.Msg) (profileSelectModel, tea.Cmd) {
	next, cmd := m.Update(msg)
	if ps, ok := next.(profileSelectModel); ok {
		return ps, cmd
	}
	return m, cmd
}

func (m profileSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = moveCursor(m.cursor, -1, len(m.profiles))
		case "down", "j":
			m.cursor = moveCursor(m.cursor, +1, len(m.profiles))
		case "enter":
			chosen := m.profiles[m.cursor]
			return m, func() tea.Msg { return profileChosenMsg{profile: chosen} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m profileSelectModel) View() tea.View {
	t := m.theme
	var sb strings.Builder
	sb.WriteString(t.TitleStyle().Render("EKVS — Select profile"))
	sb.WriteString("\n\n")
	for i, p := range m.profiles {
		line := fmt.Sprintf("%s  (%s)", p.Name, p.ServerURL)
		if i == m.cursor {
			sb.WriteString(t.SelectedMenuItemStyle().Render("> " + line))
		} else {
			sb.WriteString(t.MenuItemStyle().Render("  " + line))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(m.footer.View("↑/↓ navigate • Enter select • q quit"))
	return tea.NewView(sb.String())
}
