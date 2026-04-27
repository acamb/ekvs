// Package footer provides a stateless keyboard-hints bar for Bubble Tea v2 screens.
package footer

import "ekvs/internal/tui/theme"

// Model holds the theme used to style the footer.
// It is stateless: no Init or Update is needed.
type Model struct {
	theme theme.Theme
}

// New creates a footer Model using the given theme.
func New(t theme.Theme) Model {
	return Model{theme: t}
}

// View renders the hints string using FooterStyle.
// An empty hints string produces an empty (but styled) string.
func (m Model) View(hints string) string {
	return m.theme.FooterStyle().Render(hints)
}
