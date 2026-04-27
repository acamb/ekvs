// Package wizard implements the interactive first-run wizard for the EKVS TUI.
// It collects the data needed to create a new connection profile and optionally
// saves it to a YAML file.
package wizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"ekvs/internal/tui/config"
	"ekvs/internal/tui/footer"
	"ekvs/internal/tui/theme"
)

// DoneMsg is emitted when the wizard completes successfully.
// The root model listens for this to transition to the main screen.
type DoneMsg struct {
	Profile config.Profile
}

// saveResultMsg is an internal message carrying the result of the async file save.
type saveResultMsg struct{ err error }

// textInput is a simple text field compatible with bubbletea v2.
type textInput struct {
	value       string
	placeholder string
}

func newTextInput(placeholder, defaultVal string) textInput {
	return textInput{placeholder: placeholder, value: defaultVal}
}

func (ti textInput) view(focused bool) string {
	cursor := ""
	if focused {
		cursor = "█"
	}
	if ti.value == "" && !focused {
		return ti.placeholder
	}
	return ti.value + cursor
}

func (ti textInput) update(msg tea.KeyPressMsg) textInput {
	switch msg.String() {
	case "backspace":
		if len(ti.value) > 0 {
			runes := []rune(ti.value)
			ti.value = string(runes[:len(runes)-1])
		}
	default:
		if text := msg.Key().Text; text != "" {
			ti.value += text
		}
	}
	return ti
}

type step int

const (
	stepName step = iota
	stepServerURL
	stepIdentityFile
	stepConfirmSave
	stepFilename
)

// Model is the bubbletea model for the wizard screen.
// It is exported so the root model can embed it directly.
type Model struct {
	step           step
	name           textInput
	url            textInput
	identity       textInput
	filename       textInput
	wantSave       bool
	pendingProfile config.Profile // set by finish(), used by saveResultMsg handler
	err            string
	theme          theme.Theme
	footer         footer.Model
}

// NewModel creates a new wizard Model with the given theme.
func NewModel(t theme.Theme) Model {
	def := config.DefaultProfile()
	return Model{
		step:     stepName,
		name:     newTextInput("e.g. local", ""),
		url:      newTextInput("", def.ServerURL),
		identity: newTextInput("", def.IdentityFile),
		filename: newTextInput("", "ekvs-tui.yaml"),
		theme:    t,
		footer:   footer.New(t),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// UpdateTyped is like Update but returns the concrete Model type directly,
// avoiding the need for a type assertion in the caller.
func (m Model) UpdateTyped(msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	if nm, ok := next.(Model); ok {
		return nm, cmd
	}
	return m, cmd
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle async save result.
	if r, ok := msg.(saveResultMsg); ok {
		if r.err != nil {
			m.err = fmt.Sprintf("save failed: %v", r.err)
			return m, nil
		}
		return m, func() tea.Msg { return DoneMsg{Profile: m.pendingProfile} }
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch m.step {
		case stepName, stepServerURL, stepIdentityFile:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				var val string
				switch m.step {
				case stepName:
					val = strings.TrimSpace(m.name.value)
				case stepServerURL:
					val = strings.TrimSpace(m.url.value)
				case stepIdentityFile:
					val = strings.TrimSpace(m.identity.value)
				}
				if m.step == stepName && val == "" {
					m.err = "profile name cannot be empty"
					return m, nil
				}
				m.err = ""
				m.step++
				return m, nil
			default:
				switch m.step {
				case stepName:
					m.name = m.name.update(msg)
				case stepServerURL:
					m.url = m.url.update(msg)
				case stepIdentityFile:
					m.identity = m.identity.update(msg)
				}
			}

		case stepConfirmSave:
			switch msg.String() {
			case "y", "Y":
				m.wantSave = true
				m.step = stepFilename
			case "n", "N", "enter":
				m.wantSave = false
				m, cmd := m.finish()
				return m, cmd
			case "ctrl+c":
				return m, tea.Quit
			}

		case stepFilename:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				if strings.TrimSpace(m.filename.value) == "" {
					m.filename.value = "ekvs-tui.yaml"
				}
				m, cmd := m.finish()
				return m, cmd
			default:
				m.filename = m.filename.update(msg)
			}
		}
	}
	return m, nil
}

// buildProfile constructs a Profile from raw (possibly empty) input strings,
// falling back to DefaultProfile values where the user left fields blank.
func buildProfile(name, url, identity string) config.Profile {
	def := config.DefaultProfile()
	if u := strings.TrimSpace(url); u != "" {
		url = u
	} else {
		url = def.ServerURL
	}
	if id := strings.TrimSpace(identity); id != "" {
		identity = id
	} else {
		identity = def.IdentityFile
	}
	return config.Profile{
		Name:         strings.TrimSpace(name),
		ServerURL:    url,
		IdentityFile: identity,
		Theme:        def.Theme,
	}
}

// finish builds the profile and either emits DoneMsg directly (no save)
// or launches an async goroutine to save the config file first.
func (m Model) finish() (Model, tea.Cmd) {
	m.pendingProfile = buildProfile(m.name.value, m.url.value, m.identity.value)

	if !m.wantSave {
		profile := m.pendingProfile
		return m, func() tea.Msg { return DoneMsg{Profile: profile} }
	}

	filename := strings.TrimSpace(m.filename.value)
	if filename == "" {
		filename = "ekvs-tui.yaml"
	}
	profile := m.pendingProfile
	return m, func() tea.Msg {
		cf := &config.ConfigFile{Profiles: []config.Profile{profile}}
		return saveResultMsg{err: config.Save(filename, cf)}
	}
}

// renderSummary returns a string listing the fields that have already been
// confirmed, so each View() case only needs to render the active field.
func (m Model) renderSummary() string {
	var sb strings.Builder
	if m.step > stepName {
		fmt.Fprintf(&sb, "Profile: %s\n", m.name.value)
	}
	if m.step > stepServerURL {
		fmt.Fprintf(&sb, "Server:  %s\n", m.url.value)
	}
	if m.step > stepIdentityFile {
		fmt.Fprintf(&sb, "Key:     %s\n", m.identity.value)
	}
	return sb.String()
}

// View implements tea.Model.
func (m Model) View() tea.View {
	t := m.theme
	var sb strings.Builder

	sb.WriteString(t.TitleStyle().Render("EKVS — Initial setup"))
	sb.WriteString("\n\n")
	sb.WriteString(m.renderSummary())

	switch m.step {
	case stepName:
		sb.WriteString("Profile name:\n")
		sb.WriteString("  " + m.name.view(true) + "\n")
	case stepServerURL:
		sb.WriteString("\nServer URL:\n")
		sb.WriteString("  " + m.url.view(true) + "\n")
	case stepIdentityFile:
		sb.WriteString("\nSSH identity file path:\n")
		sb.WriteString("  " + m.identity.view(true) + "\n")
	case stepConfirmSave:
		sb.WriteString("\nSave configuration to file? [y/N] ")
	case stepFilename:
		sb.WriteString("\nConfiguration file name:\n")
		sb.WriteString("  " + m.filename.view(true) + "\n")
	}

	if m.err != "" {
		sb.WriteString("\n")
		sb.WriteString(t.ErrorStyle().Render(m.err))
	}

	sb.WriteString("\n\n")
	sb.WriteString(m.footer.View("Enter confirm • Ctrl+C cancel"))
	return tea.NewView(sb.String())
}
