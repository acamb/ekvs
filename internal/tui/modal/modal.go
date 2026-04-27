// Package modal provides a blocking error dialog for Bubble Tea v2 screens.
package modal

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"ekvs/internal/tui/theme"
)

// DismissMsg is emitted when the user dismisses the modal (Enter or Esc).
type DismissMsg struct{}

// Model is the error modal sub-model.
// It is not a full tea.Model; parent models embed it and delegate messages
// to it while the modal is active.
type Model struct {
	message string
	theme   theme.Theme
}

// New creates an error modal with the given message.
func New(t theme.Theme, message string) Model {
	return Model{theme: t, message: message}
}

// Init is a no-op; the modal has no asynchronous initialisation.
func (m Model) Init() tea.Cmd { return nil }

// Update handles Enter and Esc, emitting DismissMsg on either.
// All other keys are silently ignored (the modal is blocking).
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "enter", "esc":
			return m, func() tea.Msg { return DismissMsg{} }
		}
	}
	return m, nil
}

// View renders the modal as a centred box.
// width is the available terminal width; pass 0 to use a default of 60 columns.
func (m Model) View(width int) string {
	if width <= 0 {
		width = 60
	}

	// Inner content width: modal box uses ~70 % of available space, min 30.
	boxWidth := width * 70 / 100
	if boxWidth < 30 {
		boxWidth = 30
	}
	// Inner text width accounts for the padding (2 chars each side) added by ModalStyle.
	innerWidth := boxWidth - 8
	if innerWidth < 10 {
		innerWidth = 10
	}

	title := " Error "
	body := wordWrap(m.message, innerWidth)
	hint := "[ Press Enter to dismiss ]"

	content := strings.Join([]string{
		m.theme.TableHeaderStyle().Render(title),
		"",
		body,
		"",
		m.theme.FooterStyle().MarginTop(0).Render(hint),
	}, "\n")

	box := m.theme.ModalStyle().Width(boxWidth).Render(content)

	// Centre the box horizontally.
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, box)
}

// wordWrap breaks s into lines of at most maxWidth runes each, splitting on
// space boundaries where possible.
func wordWrap(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return s
	}

	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len([]rune(current))+1+len([]rune(w)) <= maxWidth {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n")
}
