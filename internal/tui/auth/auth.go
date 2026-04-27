package auth

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/footer"
	"ekvs/internal/tui/modal"
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

// tryLoadMsg is used internally to carry the result of the initial key-load attempt.
type tryLoadMsg struct {
	success *AuthSuccessMsg
	err     error
}

// Model is the bubbletea model for the passphrase prompt screen.
type Model struct {
	pemBytes   []byte
	input      string
	theme      theme.Theme
	showModal  bool
	modalModel modal.Model
	footer     footer.Model
}

// New creates a new auth Model that will try to load the SSH key at identityFile.
// identityFile may use ~ for the home directory.
func New(identityFile string, t theme.Theme) Model {
	return Model{
		pemBytes: loadPEM(identityFile),
		theme:    t,
		footer:   footer.New(t),
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
	// When the error modal is active, route all input to it.
	if m.showModal {
		if _, ok := msg.(modal.DismissMsg); ok {
			m.showModal = false
			m.input = ""
			return m, nil
		}
		updated, cmd := m.modalModel.Update(msg)
		m.modalModel = updated
		return m, cmd
	}

	switch msg := msg.(type) {
	case tryLoadMsg:
		if msg.success != nil {
			return m, func() tea.Msg { return *msg.success }
		}
		if msg.err != nil {
			m.modalModel = modal.New(m.theme, msg.err.Error())
			m.showModal = true
		}
		// else: no error → passphrase required, stay at prompt
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return AuthCancelMsg{} }

		case "enter":
			signer, pub, err := internalssh.ParsePrivateKeyWithPassphrase(m.pemBytes, []byte(m.input))
			if err != nil {
				m.modalModel = modal.New(m.theme, "wrong passphrase — press Enter to retry or Esc to cancel")
				m.showModal = true
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
			if text := msg.Key().Text; text != "" {
				m.input += text
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
	sb.WriteString(m.theme.MenuItemStyle().Render("Enter passphrase: "))
	sb.WriteString(strings.Repeat("*", len([]rune(m.input))))
	sb.WriteString("█\n\n")

	if m.showModal {
		sb.WriteString(m.modalModel.View(0))
	} else {
		sb.WriteString(m.footer.View("Enter confirm • Esc cancel"))
	}

	return tea.NewView(sb.String())
}
