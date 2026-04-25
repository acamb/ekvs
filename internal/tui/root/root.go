// Package root implements the top-level bubbletea model for the EKVS TUI.
// It owns the full lifecycle: wizard → profile selection → main menu,
// all within a single tea.Program so the terminal is initialised only once.
package root

import (
	tea "charm.land/bubbletea/v2"

	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/theme"
	"ekvs/internal/tui/wizard"
)

// screen identifies which "page" is currently active.
type screen int

const (
	screenWizard screen = iota
	screenProfileSelect
	screenMain
)

// moveCursor returns a new cursor position after moving delta steps (+1 or -1)
// within a list of length n, wrapping around at both ends.
func moveCursor(cursor, delta, n int) int {
	return (cursor + delta + n) % n
}

// Model is the single top-level bubbletea model.
type Model struct {
	screen  screen
	theme   theme.Theme
	profile tuiconfig.Profile

	wizard        wizard.Model
	profileSelect profileSelectModel
	main          mainModel
}

// New creates the root model.
//
//   - cfg is nil when no configuration file exists (wizard will run).
//   - cfg.Profiles has one entry → skip profile selection.
//   - cfg.Profiles has more entries → show profile selection screen.
func New(cfg *tuiconfig.ConfigFile, defaultTheme theme.Theme) Model {
	m := Model{theme: defaultTheme}

	switch {
	case cfg == nil || len(cfg.Profiles) == 0:
		m.screen = screenWizard
		m.wizard = wizard.NewModel(defaultTheme)

	case len(cfg.Profiles) == 1:
		p := cfg.Profiles[0]
		t := resolveTheme(p.Theme, defaultTheme)
		m.screen = screenMain
		m.profile = p
		m.theme = t
		m.main = newMainModel(t)

	default:
		m.screen = screenProfileSelect
		m.profileSelect = newProfileSelectModel(cfg.Profiles, defaultTheme)
	}

	return m
}

func resolveTheme(name string, fallback theme.Theme) theme.Theme {
	t, err := theme.NewTheme(name)
	if err != nil {
		return fallback
	}
	return t
}

// Init implements tea.Model. Delegates to the active sub-model so that any
// initial commands it emits are honoured from the start.
func (m Model) Init() tea.Cmd {
	switch m.screen {
	case screenWizard:
		return m.wizard.Init()
	case screenProfileSelect:
		return m.profileSelect.Init()
	case screenMain:
		return m.main.Init()
	}
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle cross-screen transition messages first.
	switch msg := msg.(type) {
	case wizard.DoneMsg:
		t := resolveTheme(msg.Profile.Theme, m.theme)
		m.theme = t
		m.profile = msg.Profile
		m.screen = screenMain
		m.main = newMainModel(t)
		return m, nil

	case profileChosenMsg:
		t := resolveTheme(msg.profile.Theme, m.theme)
		m.theme = t
		m.profile = msg.profile
		m.screen = screenMain
		m.main = newMainModel(t)
		return m, nil
	}

	// Delegate to the active screen.
	// Sub-models expose *Typed() helpers that return their concrete type
	// directly, so there is no silent type-assertion failure.
	switch m.screen {
	case screenWizard:
		wm, cmd := m.wizard.UpdateTyped(msg)
		m.wizard = wm
		return m, cmd

	case screenProfileSelect:
		ps, cmd := m.profileSelect.updateTyped(msg)
		m.profileSelect = ps
		return m, cmd

	case screenMain:
		mm, cmd := m.main.updateTyped(msg)
		m.main = mm
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	switch m.screen {
	case screenWizard:
		return m.wizard.View()
	case screenProfileSelect:
		return m.profileSelect.View()
	case screenMain:
		return m.main.View()
	}
	return tea.NewView("")
}
