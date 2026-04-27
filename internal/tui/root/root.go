// Package root implements the top-level bubbletea model for the EKVS TUI.
// It owns the full lifecycle: wizard → profile selection → main menu,
// all within a single tea.Program so the terminal is initialised only once.
package root

import (
	"crypto"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"

	"ekvs/internal/tui/auth"
	"ekvs/internal/tui/client"
	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/profiles"
	"ekvs/internal/tui/projects"
	"ekvs/internal/tui/secrets"
	"ekvs/internal/tui/session"
	"ekvs/internal/tui/theme"
	"ekvs/internal/tui/wizard"

	gossh "golang.org/x/crypto/ssh"
)

// screen identifies which "page" is currently active.
type screen int

const (
	screenWizard screen = iota
	screenProfileSelect
	screenMain
	screenAuth
	screenProjects
	screenSecrets
	screenProfiles
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
	session session.Session

	// Terminal dimensions — updated on every tea.WindowSizeMsg.
	width  int
	height int

	// config holds the in-memory ConfigFile; may be nil before wizard completes.
	config     *tuiconfig.ConfigFile
	configPath string

	wizard        wizard.Model
	profileSelect profileSelectModel
	main          mainModel
	authModel     auth.Model
	projectsModel projects.Model
	secretsModel  secrets.Model
	profilesModel profiles.Model
	pendingScreen screen

	// Error modal overlay (shown on top of screenMain when auth key-derivation fails).
	showModal  bool
	modalModel modal.Model
}

// New creates the root model.
//
//   - cfg is nil when no configuration file exists (wizard will run).
//   - cfg.Profiles has one entry → skip profile selection.
//   - cfg.Profiles has more entries → show profile selection screen.
func New(cfg *tuiconfig.ConfigFile, configPath string, defaultTheme theme.Theme) Model {
	m := Model{theme: defaultTheme, config: cfg, configPath: configPath}

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
	case screenAuth:
		return m.authModel.Init()
	case screenProjects:
		return m.projectsModel.Init()
	case screenSecrets:
		return m.secretsModel.Init()
	case screenProfiles:
		return m.profilesModel.Init()
	}
	return nil
}

// newClient builds a signed HTTP client from the current profile and session.
func newClient(m Model) *client.Client {
	return client.New(m.profile.ServerURL, &m.session)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always track terminal dimensions.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		// Propagate to the active sub-model via the dispatch below.
	}

	// When the error modal is active, route all input to it.
	if m.showModal {
		if _, ok := msg.(modal.DismissMsg); ok {
			m.showModal = false
			return m, nil
		}
		updated, cmd := m.modalModel.Update(msg)
		m.modalModel = updated
		return m, cmd
	}

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

	case triggerAuthMsg:
		m.pendingScreen = msg.returnTo
		m.authModel = auth.New(m.profile.IdentityFile, m.theme)
		m.screen = screenAuth
		return m, m.authModel.Init()

	case auth.AuthSuccessMsg:
		var signer crypto.Signer
		var pub gossh.PublicKey
		if msg.Signer != nil {
			signer = msg.Signer.(crypto.Signer)
		}
		if msg.PublicKey != nil {
			pub = msg.PublicKey.(gossh.PublicKey)
		}
		if err := m.session.SetAuthenticated(signer, pub, msg.Fingerprint); err != nil {
			// Key derivation failed (unsupported key type). Show error modal over main menu.
			m.screen = screenMain
			m.modalModel = modal.New(m.theme, fmt.Sprintf("authentication error: %v", err))
			m.showModal = true
			return m, nil
		}
		m.screen = m.pendingScreen
		if m.pendingScreen == screenProjects {
			m.projectsModel = projects.New(newClient(m), m.theme)
			return m, m.projectsModel.Init()
		}
		return m, nil

	case auth.AuthCancelMsg:
		m.screen = screenMain
		return m, nil

	case profileSwitchMsg:
		m.session.Clear()
		t := resolveTheme(msg.profile.Theme, m.theme)
		m.theme = t
		m.profile = msg.profile
		m.screen = screenMain
		m.main = newMainModel(t)
		return m, nil

	case triggerProjectsMsg:
		if !m.session.IsAuthenticated() {
			m.pendingScreen = screenProjects
			m.authModel = auth.New(m.profile.IdentityFile, m.theme)
			m.screen = screenAuth
			return m, m.authModel.Init()
		}
		m.projectsModel = projects.New(newClient(m), m.theme)
		m.screen = screenProjects
		return m, m.projectsModel.Init()

	case projects.BackMsg:
		m.screen = screenMain
		return m, nil

	case projects.OpenSecretsMsg:
		m.secretsModel = secrets.New(msg.Project, newClient(m), &m.session, m.theme)
		m.screen = screenSecrets
		return m, m.secretsModel.Init()

	case secrets.BackMsg:
		m.projectsModel = projects.New(newClient(m), m.theme)
		m.screen = screenProjects
		return m, m.projectsModel.Init()

	case triggerProfilesMsg:
		cfg := m.config
		if cfg == nil {
			cfg = &tuiconfig.ConfigFile{}
		}
		m.profilesModel = profiles.New(cfg, m.configPath, m.profile.Name, m.theme)
		if m.width > 0 {
			m.profilesModel, _ = m.profilesModel.UpdateTyped(
				tea.WindowSizeMsg{Width: m.width, Height: m.height},
			)
		}
		m.screen = screenProfiles
		return m, m.profilesModel.Init()

	case profiles.BackMsg:
		m.screen = screenMain
		return m, nil

	case profiles.SwitchMsg:
		m.session.Clear()
		t := resolveTheme(msg.Profile.Theme, m.theme)
		m.theme = t
		m.profile = msg.Profile
		m.screen = screenMain
		m.main = newMainModel(t)
		return m, nil

	case profiles.ReloadActiveMsg:
		m.session.Clear()
		if msg.Config != nil {
			m.config = msg.Config
		}
		t := resolveTheme(msg.Profile.Theme, m.theme)
		m.theme = t
		m.profile = msg.Profile
		m.screen = screenMain
		m.main = newMainModel(t)
		return m, nil

	case profiles.ConfigChangedMsg:
		m.config = msg.Config
		if msg.ActiveDeleted {
			if len(msg.Config.Profiles) == 0 {
				// No profiles left → wizard
				m.screen = screenWizard
				m.wizard = wizard.NewModel(m.theme)
				return m, m.wizard.Init()
			}
			// Remaining profiles → profile selection
			m.screen = screenProfileSelect
			m.profileSelect = newProfileSelectModel(msg.Config.Profiles, m.theme)
			return m, m.profileSelect.Init()
		}
		// Non-active profile changed: stay on profiles screen with updated config
		m.profilesModel = profiles.New(msg.Config, m.configPath, m.profile.Name, m.theme)
		if m.width > 0 {
			m.profilesModel, _ = m.profilesModel.UpdateTyped(
				tea.WindowSizeMsg{Width: m.width, Height: m.height},
			)
		}
		m.screen = screenProfiles
		return m, m.profilesModel.Init()
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

	case screenAuth:
		am, cmd := m.authModel.UpdateTyped(msg)
		m.authModel = am
		return m, cmd

	case screenProjects:
		pm, cmd := m.projectsModel.UpdateTyped(msg)
		m.projectsModel = pm
		return m, cmd

	case screenSecrets:
		sm, cmd := m.secretsModel.UpdateTyped(msg)
		m.secretsModel = sm
		return m, cmd

	case screenProfiles:
		pm, cmd := m.profilesModel.UpdateTyped(msg)
		m.profilesModel = pm
		return m, cmd
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var content string
	switch m.screen {
	case screenWizard:
		content = m.wizard.View().Content
	case screenProfileSelect:
		content = m.profileSelect.View().Content
	case screenMain:
		content = m.main.View().Content
	case screenAuth:
		content = m.authModel.View().Content
	case screenProjects:
		content = m.projectsModel.View().Content
	case screenSecrets:
		content = m.secretsModel.View().Content
	case screenProfiles:
		content = m.profilesModel.View().Content
	}

	if m.showModal {
		content += "\n" + m.modalModel.View(m.width)
	}

	// Guard against pre-first-WindowSizeMsg: do not apply background fill
	// before we know the terminal dimensions (would produce a 0×0 box).
	if m.width == 0 || m.height == 0 {
		return tea.NewView(content)
	}

	filled := lipgloss.NewStyle().
		Background(m.theme.BackgroundColor()).
		Width(m.width).
		Height(m.height).
		Render(content)

	return tea.NewView(filled)
}
