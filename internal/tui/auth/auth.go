package auth

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/theme"
)

// AuthSuccessMsg is emitted when authentication succeeds.
type AuthSuccessMsg struct {
	Signer      interface{} // crypto.Signer
	PublicKey   interface{} // gossh.PublicKey
	Fingerprint string
}

// AuthCancelMsg is emitted when the user cancels authentication.
type AuthCancelMsg struct{}

type authState int

const (
	statePrompt authState = iota
	stateError
)

// tryLoadMsg is used internally to carry the result of the initial key-load attempt.
type tryLoadMsg struct {
	success *AuthSuccessMsg
	err     error
}

// Model is the bubbletea model for the passphrase prompt screen.
type Model struct {
	pemBytes []byte
	state    authState
	input    string
	errMsg   string
	theme    theme.Theme
}

// New creates a new auth Model that will try to load the SSH key at identityFile.
// identityFile may use ~ for the home directory.
func New(identityFile string, t theme.Theme) Model {
	return Model{
		pemBytes: loadPEM(identityFile),
		theme:    t,
	}
}

func loadPEM(path string) []byte {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = home + path[1:]
		}
	}
	b, _ := os.ReadFile(path)
	return b
}

// Init attempts to parse the key without a passphrase.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		if len(m.pemBytes) == 0 {
			return tryLoadMsg{err: fmt.Errorf("identity file not found or unreadable")}
		}
		signer, pub, err := internalssh.ParsePrivateKey(m.pemBytes)
		if err == internalssh.ErrPassphraseRequired {
			return tryLoadMsg{} // no success, no error → show prompt
		}
		if err != nil {
			return tryLoadMsg{err: err}
		}
		return tryLoadMsg{success: &AuthSuccessMsg{
			Signer:      signer,
			PublicKey:   pub,
			Fingerprint: internalssh.Fingerprint(pub),
		}}
	}
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
	case tryLoadMsg:
		if msg.success != nil {
			return m, func() tea.Msg { return *msg.success }
		}
		if msg.err != nil {
			m.state = stateError
			m.errMsg = msg.err.Error()
		}
		// else: no error → passphrase required, stay in statePrompt
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return AuthCancelMsg{} }

		case "enter":
			if m.state == stateError {
				// Retry
				m.state = statePrompt
				m.input = ""
				m.errMsg = ""
				return m, m.Init()
			}
			// Submit passphrase
			signer, pub, err := internalssh.ParsePrivateKeyWithPassphrase(m.pemBytes, []byte(m.input))
			if err != nil {
				m.state = stateError
				m.errMsg = "wrong passphrase, press Enter to retry or Esc to cancel"
				return m, nil
			}
			return m, func() tea.Msg {
				return AuthSuccessMsg{
					Signer:      signer,
					PublicKey:   pub,
					Fingerprint: internalssh.Fingerprint(pub),
				}
			}

		case "backspace":
			if len(m.input) > 0 {
				r := []rune(m.input)
				m.input = string(r[:len(r)-1])
			}

		default:
			if m.state == statePrompt {
				if text := msg.Key().Text; text != "" {
					m.input += text
				}
			}
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var sb strings.Builder
	sb.WriteString(m.theme.TitleStyle().Render("Authentication"))
	sb.WriteString("\n\n")

	switch m.state {
	case statePrompt:
		sb.WriteString(m.theme.MenuItemStyle().Render("Enter passphrase: "))
		sb.WriteString(strings.Repeat("*", len([]rune(m.input))))
		sb.WriteString("█\n\n")
		sb.WriteString(m.theme.StatusBarStyle().Render("Enter confirm • Esc cancel"))

	case stateError:
		sb.WriteString(m.theme.ErrorStyle().Render(m.errMsg))
		sb.WriteString("\n\n")
		sb.WriteString(m.theme.StatusBarStyle().Render("Enter retry • Esc cancel"))
	}

	return tea.NewView(sb.String())
}
