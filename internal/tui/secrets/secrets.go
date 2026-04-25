package secrets

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"golang.design/x/clipboard"

	"ekvs/internal/tui/client"
	"ekvs/internal/tui/session"
	"ekvs/internal/tui/theme"
)

const pageSize = 10

// mode represents the current interaction mode of the Secrets screen.
type mode int

const (
	modeList   mode = iota
	modeAdd         // two-field input: key then value
	modeEdit        // value field only (key is read-only)
	modeDelete      // inline y/n confirmation
)

// apiClient is the subset of client.Client used by this model.
// Defined as an interface to allow test fakes.
type apiClient interface {
	ListSecrets(project string) ([]client.SecretEntry, error)
	SetSecret(project, key, value string) error
	DeleteSecret(project, key string) error
}

// Model is the bubbletea model for the Secrets screen.
type Model struct {
	project     string
	client      apiClient
	sess        *session.Session
	theme       theme.Theme
	secrets     []client.SecretEntry
	cursor      int
	page        int
	mode        mode
	inputKey    string
	inputValue  string
	activeField int // 0 = key field, 1 = value field (in modeAdd/modeEdit)
	err         error
	loading     bool
}

// New creates a Model ready to be initialised.
func New(project string, c *client.Client, sess *session.Session, t theme.Theme) Model {
	return Model{project: project, client: c, sess: sess, theme: t}
}

// newWithClient creates a Model using any apiClient implementation (used in tests).
func newWithClient(project string, c apiClient, sess *session.Session, t theme.Theme) Model {
	return Model{project: project, client: c, sess: sess, theme: t}
}

// ── commands ──────────────────────────────────────────────────────────────────

func (m Model) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		secrets, err := m.client.ListSecrets(m.project)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("fetch secrets: %w", err)}
		}
		return FetchedMsg{Secrets: secrets}
	}
}

func (m Model) setSecretCmd(key, plaintext string) tea.Cmd {
	return func() tea.Msg {
		blob, err := m.sess.Encrypt(plaintext)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("encrypt: %w", err)}
		}
		if err := m.client.SetSecret(m.project, key, blob); err != nil {
			return ErrMsg{Err: fmt.Errorf("set secret: %w", err)}
		}
		// Re-fetch after successful set.
		secrets, err := m.client.ListSecrets(m.project)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("fetch secrets: %w", err)}
		}
		return FetchedMsg{Secrets: secrets}
	}
}

func (m Model) deleteSecretCmd(key string) tea.Cmd {
	return func() tea.Msg {
		if err := m.client.DeleteSecret(m.project, key); err != nil {
			return ErrMsg{Err: fmt.Errorf("delete secret: %w", err)}
		}
		secrets, err := m.client.ListSecrets(m.project)
		if err != nil {
			return ErrMsg{Err: fmt.Errorf("fetch secrets: %w", err)}
		}
		return FetchedMsg{Secrets: secrets}
	}
}

// ── pagination helpers ────────────────────────────────────────────────────────

func (m Model) totalPages() int {
	if len(m.secrets) == 0 {
		return 1
	}
	return (len(m.secrets) + pageSize - 1) / pageSize
}

func (m Model) pageSecrets() []client.SecretEntry {
	start := m.page * pageSize
	if start >= len(m.secrets) {
		return nil
	}
	end := start + pageSize
	if end > len(m.secrets) {
		end = len(m.secrets)
	}
	return m.secrets[start:end]
}

func (m Model) selectedEntry() (client.SecretEntry, bool) {
	items := m.pageSecrets()
	if len(items) == 0 || m.cursor >= len(items) {
		return client.SecretEntry{}, false
	}
	return items[m.cursor], true
}

// ── Init / Update / View ──────────────────────────────────────────────────────

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	m.loading = true
	return m.fetchCmd()
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

	case FetchedMsg:
		m.secrets = msg.Secrets
		m.loading = false
		m.err = nil
		if m.cursor >= len(m.pageSecrets()) {
			m.cursor = 0
		}
		m.mode = modeList
		return m, nil

	case ErrMsg:
		m.loading = false
		m.err = msg.Err
		return m, nil

	case tea.KeyPressMsg:
		// Clear previous error on any keypress.
		m.err = nil

		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeAdd:
			return m.updateAdd(msg)
		case modeEdit:
			return m.updateEdit(msg)
		case modeDelete:
			return m.updateDelete(msg)
		}
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.pageSecrets()
	switch msg.String() {
	case "up", "k":
		if len(items) > 0 {
			m.cursor = (m.cursor - 1 + len(items)) % len(items)
		}
	case "down", "j":
		if len(items) > 0 {
			m.cursor = (m.cursor + 1) % len(items)
		}
	case "right":
		if m.page < m.totalPages()-1 {
			m.page++
			m.cursor = 0
		}
	case "left":
		if m.page > 0 {
			m.page--
			m.cursor = 0
		}
	case "n":
		m.mode = modeAdd
		m.inputKey = ""
		m.inputValue = ""
		m.activeField = 0
	case "e":
		if entry, ok := m.selectedEntry(); ok {
			decrypted, err := m.sess.Decrypt(entry.Value)
			if err != nil {
				m.err = fmt.Errorf("decrypt: %w", err)
				return m, nil
			}
			m.mode = modeEdit
			m.inputKey = entry.Key
			m.inputValue = decrypted
			m.activeField = 1
		}
	case "d":
		if len(items) > 0 {
			m.mode = modeDelete
		}
	case "c":
		if entry, ok := m.selectedEntry(); ok {
			decrypted, err := m.sess.Decrypt(entry.Value)
			if err != nil {
				m.err = fmt.Errorf("clipboard: %w", err)
				return m, nil
			}
			if initErr := clipboard.Init(); initErr != nil {
				m.err = fmt.Errorf("clipboard unavailable: %w", initErr)
				return m, nil
			}
			clipboard.Write(clipboard.FmtText, []byte(decrypted))
		}
	case "esc", "q":
		return m, func() tea.Msg { return BackMsg{} }
	}
	return m, nil
}

func (m Model) updateAdd(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.inputKey = ""
		m.inputValue = ""
		m.activeField = 0
	case "tab", "enter":
		if m.activeField == 0 {
			// Advance to value field.
			if strings.TrimSpace(m.inputKey) == "" {
				m.err = fmt.Errorf("key cannot be empty")
				return m, nil
			}
			m.activeField = 1
		} else {
			// Submit.
			key := strings.TrimSpace(m.inputKey)
			value := m.inputValue
			if value == "" {
				m.err = fmt.Errorf("value cannot be empty")
				return m, nil
			}
			m.loading = true
			m.inputKey = ""
			m.inputValue = ""
			m.activeField = 0
			return m, m.setSecretCmd(key, value)
		}
	case "backspace":
		if m.activeField == 0 {
			if len(m.inputKey) > 0 {
				r := []rune(m.inputKey)
				m.inputKey = string(r[:len(r)-1])
			}
		} else {
			if len(m.inputValue) > 0 {
				r := []rune(m.inputValue)
				m.inputValue = string(r[:len(r)-1])
			}
		}
	default:
		if text := msg.Key().Text; text != "" {
			if m.activeField == 0 {
				m.inputKey += text
			} else {
				m.inputValue += text
			}
		}
	}
	return m, nil
}

func (m Model) updateEdit(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.inputKey = ""
		m.inputValue = ""
		m.activeField = 0
	case "enter":
		key := m.inputKey
		value := m.inputValue
		if value == "" {
			m.err = fmt.Errorf("value cannot be empty")
			return m, nil
		}
		m.loading = true
		m.inputKey = ""
		m.inputValue = ""
		m.activeField = 0
		return m, m.setSecretCmd(key, value)
	case "backspace":
		if len(m.inputValue) > 0 {
			r := []rune(m.inputValue)
			m.inputValue = string(r[:len(r)-1])
		}
	default:
		if text := msg.Key().Text; text != "" {
			m.inputValue += text
		}
	}
	return m, nil
}

func (m Model) updateDelete(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		entry, ok := m.selectedEntry()
		if !ok {
			m.mode = modeList
			return m, nil
		}
		m.loading = true
		m.mode = modeList
		return m, m.deleteSecretCmd(entry.Key)
	default:
		m.mode = modeList
	}
	return m, nil
}

// decryptedValue attempts to decrypt entry.Value using the session.
// Returns the decrypted string or "<error>" on failure.
func (m Model) decryptedValue(entry client.SecretEntry) string {
	pt, err := m.sess.Decrypt(entry.Value)
	if err != nil {
		return "<error>"
	}
	return pt
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var sb strings.Builder

	sb.WriteString(m.theme.TitleStyle().Render(fmt.Sprintf("Project: %s", m.project)))
	sb.WriteString("\n")

	if m.loading {
		sb.WriteString(m.theme.MenuItemStyle().Render("  Loading…"))
		sb.WriteString("\n")
		return tea.NewView(sb.String())
	}

	items := m.pageSecrets()

	switch m.mode {
	case modeList:
		if len(items) == 0 {
			sb.WriteString(m.theme.MenuItemStyle().Render("  (no secrets)"))
			sb.WriteString("\n")
		} else {
			// Compute max key width for alignment.
			maxKey := 3 // minimum "KEY"
			for _, e := range items {
				if len(e.Key) > maxKey {
					maxKey = len(e.Key)
				}
			}
			header := fmt.Sprintf("  %-*s  VALUE (decrypted)", maxKey, "KEY")
			sb.WriteString(m.theme.StatusBarStyle().Render(header))
			sb.WriteString("\n")
			for i, e := range items {
				val := m.decryptedValue(e)
				line := fmt.Sprintf("  %-*s  %s", maxKey, e.Key, val)
				if i == m.cursor {
					sb.WriteString(m.theme.SelectedMenuItemStyle().Render(">" + line[1:]))
				} else {
					sb.WriteString(m.theme.MenuItemStyle().Render(line))
				}
				sb.WriteString("\n")
			}
		}

	case modeAdd:
		sb.WriteString(m.theme.MenuItemStyle().Render("  Add secret"))
		sb.WriteString("\n")
		keyLine := fmt.Sprintf("  KEY:   %s", m.inputKey)
		valLine := fmt.Sprintf("  VALUE: %s", m.inputValue)
		if m.activeField == 0 {
			keyLine += "█"
		} else {
			valLine += "█"
		}
		sb.WriteString(m.theme.MenuItemStyle().Render(keyLine))
		sb.WriteString("\n")
		sb.WriteString(m.theme.MenuItemStyle().Render(valLine))
		sb.WriteString("\n")

	case modeEdit:
		sb.WriteString(m.theme.MenuItemStyle().Render("  Edit secret"))
		sb.WriteString("\n")
		sb.WriteString(m.theme.MenuItemStyle().Render(fmt.Sprintf("  KEY:   %s", m.inputKey)))
		sb.WriteString("\n")
		sb.WriteString(m.theme.MenuItemStyle().Render(fmt.Sprintf("  VALUE: %s█", m.inputValue)))
		sb.WriteString("\n")

	case modeDelete:
		if entry, ok := m.selectedEntry(); ok {
			sb.WriteString(m.theme.ErrorStyle().Render(fmt.Sprintf(`  Delete "%s"? [y/N]`, entry.Key)))
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n")

	// Error line.
	if m.err != nil {
		sb.WriteString(m.theme.ErrorStyle().Render(fmt.Sprintf("  Error: %s", m.err.Error())))
		sb.WriteString("\n")
	}

	// Status bar.
	pageInfo := fmt.Sprintf("Page %d/%d", m.page+1, m.totalPages())
	var hints string
	switch m.mode {
	case modeList:
		hints = "↑/↓ navigate • ←/→ page • n add • e edit • d delete • c copy • Esc back"
	case modeAdd:
		if m.activeField == 0 {
			hints = "Tab/Enter next field • Esc cancel"
		} else {
			hints = "Enter confirm • Esc cancel"
		}
	case modeEdit:
		hints = "Enter confirm • Esc cancel"
	case modeDelete:
		hints = "y confirm • any key cancel"
	}
	sb.WriteString(m.theme.StatusBarStyle().Render(fmt.Sprintf("%s  %s", pageInfo, hints)))

	return tea.NewView(sb.String())
}
