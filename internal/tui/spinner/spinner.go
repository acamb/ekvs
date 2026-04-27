// Package spinner provides a generic animated loading indicator for Bubble Tea v2.
package spinner

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/theme"
)

// frames is the sequence of braille Unicode characters shown in rotation.
var frames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const tickInterval = 100 * time.Millisecond

// TickMsg is the internal clock tick that advances the spinner frame.
type TickMsg struct{}

// tick returns a command that fires TickMsg after tickInterval.
func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg {
		return TickMsg{}
	})
}

// Model is the spinner helper sub-model.
// It is not a tea.Model itself; parent models embed it and delegate TickMsg to it.
type Model struct {
	frame int
	theme theme.Theme
}

// New creates a spinner Model using the given theme.
func New(t theme.Theme) Model {
	return Model{theme: t}
}

// Init returns the first tick command. Call this from the parent model's Init.
func (m Model) Init() tea.Cmd {
	return tick()
}

// Update handles TickMsg and schedules the next tick.
// Call this from the parent model's Update when the message is a TickMsg.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(TickMsg); ok {
		m.frame = (m.frame + 1) % len(frames)
		return m, tick()
	}
	return m, nil
}

// View returns the current spinner frame rendered with SpinnerStyle.
func (m Model) View() string {
	return m.theme.SpinnerStyle().Render(frames[m.frame])
}
