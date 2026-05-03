package secrets

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"golang.design/x/clipboard"

	"ekvs/internal/tui/client"
	"ekvs/internal/tui/footer"
	"ekvs/internal/tui/modal"
	"ekvs/internal/tui/session"
	"ekvs/internal/tui/spinner"
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
	modeError       // blocking error modal overlay
	modeSearch      // incremental key filter
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
	searchQuery string

	table      table.Model
	spinner    spinner.Model
	footer     footer.Model
	modalModel modal.Model
}

// buildTable constructs a bubbles table model for the current page.
func (m Model) buildTable() table.Model {
	items := m.pageSecrets()

	// Compute column widths.
	keyW := 3 // minimum "KEY"
	valW := 5 // minimum "VALUE"
	for _, e := range items {
		if len(e.Key) > keyW {
			keyW = len(e.Key)
		}
		dec := m.decryptedValue(e)
		if len(dec) > valW {
			valW = len(dec)
		}
	}

	cols := []table.Column{
		{Title: "KEY", Width: keyW},
		{Title: "VALUE", Width: valW},
	}

	rows := make([]table.Row, len(items))
	for i, e := range items {
		rows[i] = table.Row{e.Key, m.decryptedValue(e)}
	}

	// Each column has padding(0,1) = 2 extra chars; account for both columns.
	totalWidth := keyW + valW + 4

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithWidth(totalWidth),
		table.WithHeight(len(rows)+1),
	)
	if len(rows) > 0 {
		t.SetCursor(m.cursor)
	}
	return t
}

// New creates a Model ready to be initialised.
func New(project string, c *client.Client, sess *session.Session, t theme.Theme) Model {
	m := Model{
		project: project,
		client:  c,
		sess:    sess,
		theme:   t,
		spinner: spinner.New(t),
		footer:  footer.New(t),
	}
	m.table = m.buildTable()
	return m
}

// newWithClient creates a Model using any apiClient implementation (used in tests).
func newWithClient(project string, c apiClient, sess *session.Session, t theme.Theme) Model {
	m := Model{
		project: project,
		client:  c,
		sess:    sess,
		theme:   t,
		spinner: spinner.New(t),
		footer:  footer.New(t),
	}
	m.table = m.buildTable()
	return m
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

func (m Model) filteredSecrets() []client.SecretEntry {
	if m.searchQuery == "" {
		return m.secrets
	}
	q := strings.ToLower(m.searchQuery)
	out := make([]client.SecretEntry, 0, len(m.secrets))
	for _, e := range m.secrets {
		if strings.Contains(strings.ToLower(e.Key), q) {
			out = append(out, e)
		}
	}
	return out
}

func (m Model) totalPages() int {
	n := len(m.filteredSecrets())
	if n == 0 {
		return 1
	}
	return (n + pageSize - 1) / pageSize
}

func (m Model) pageSecrets() []client.SecretEntry {
	all := m.filteredSecrets()
	start := m.page * pageSize
	if start >= len(all) {
		return nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end]
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
	return tea.Batch(m.fetchCmd(), m.spinner.Init())
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
	// Delegate spinner ticks.
	if _, ok := msg.(spinner.TickMsg); ok {
		s, cmd := m.spinner.Update(msg)
		m.spinner = s
		return m, cmd
	}

	// When the error modal is active, route all input to it.
	if m.mode == modeError {
		if _, ok := msg.(modal.DismissMsg); ok {
			m.mode = modeList
			m.err = nil
			return m, nil
		}
		updated, cmd := m.modalModel.Update(msg)
		m.modalModel = updated
		return m, cmd
	}

	switch msg := msg.(type) {

	case FetchedMsg:
		m.secrets = msg.Secrets
		m.loading = false
		m.err = nil
		if m.cursor >= len(m.pageSecrets()) {
			m.cursor = 0
		}
		m.mode = modeList
		m.table = m.buildTable()
		return m, nil

	case ErrMsg:
		m.loading = false
		m.err = msg.Err
		m.modalModel = modal.New(m.theme, msg.Err.Error())
		m.mode = modeError
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
		case modeSearch:
			return m.updateSearch(msg)
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
			m.table = m.buildTable()
		}
	case "down", "j":
		if len(items) > 0 {
			m.cursor = (m.cursor + 1) % len(items)
			m.table = m.buildTable()
		}
	case "right":
		if m.page < m.totalPages()-1 {
			m.page++
			m.cursor = 0
			m.table = m.buildTable()
		}
	case "left":
		if m.page > 0 {
			m.page--
			m.cursor = 0
			m.table = m.buildTable()
		}
	case "n":
		m.mode = modeAdd
		m.inputKey = ""
		m.inputValue = ""
		m.activeField = 0
	case "/":
		m.mode = modeSearch
		m.searchQuery = ""
		m.cursor = 0
		m.table = m.buildTable()
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
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.cursor = 0
			m.page = 0
			m.table = m.buildTable()
			return m, nil
		}
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
	case "ctrl+v":
		if initErr := clipboard.Init(); initErr != nil {
			m.err = fmt.Errorf("clipboard unavailable: %w", initErr)
			return m, nil
		}
		paste := clipboard.Read(clipboard.FmtText)
		if len(paste) != 0 {
			if m.activeField == 0 {
				m.inputKey += string(paste)
			} else {
				m.inputValue += string(paste)
			}
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
	case "ctrl+v":
		if initErr := clipboard.Init(); initErr != nil {
			m.err = fmt.Errorf("clipboard unavailable: %w", initErr)
			return m, nil
		}
		paste := clipboard.Read(clipboard.FmtText)
		if len(paste) != 0 {
			if m.activeField == 0 {
				m.inputKey += string(paste)
			} else {
				m.inputValue += string(paste)
			}
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

func (m Model) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.mode = modeList
	case "backspace":
		if len(m.searchQuery) > 0 {
			r := []rune(m.searchQuery)
			m.searchQuery = string(r[:len(r)-1])
			m.cursor = 0
			m.page = 0
			m.table = m.buildTable()
		}
	default:
		if text := msg.Key().Text; text != "" {
			m.searchQuery += text
			m.cursor = 0
			m.page = 0
			m.table = m.buildTable()
		}
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
		sb.WriteString(m.theme.MenuItemStyle().Render(
			fmt.Sprintf("  %s  Loading…", m.spinner.View())))
		sb.WriteString("\n")
		return tea.NewView(sb.String())
	}

	// Error modal overlay.
	if m.mode == modeError {
		sb.WriteString(m.modalModel.View(0))
		return tea.NewView(sb.String())
	}

	items := m.pageSecrets()

	switch m.mode {
	case modeList, modeSearch:
		if len(items) == 0 {
			if m.searchQuery != "" {
				sb.WriteString(m.theme.MenuItemStyle().Render(fmt.Sprintf("  (no secrets match %q)", m.searchQuery)))
			} else {
				sb.WriteString(m.theme.MenuItemStyle().Render("  (no secrets)"))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString(m.table.View())
			sb.WriteString("\n")
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

	// Footer hints.
	pageInfo := fmt.Sprintf("Page %d/%d", m.page+1, m.totalPages())
	var hints string
	switch m.mode {
	case modeList:
		if m.searchQuery != "" {
			hints = fmt.Sprintf("%s  filter:%q  ↑/↓ navigate • ←/→ page • n add • e edit • d delete • c copy • / search • Esc clear filter", pageInfo, m.searchQuery)
		} else {
			hints = fmt.Sprintf("%s  ↑/↓ navigate • ←/→ page • n add • e edit • d delete • c copy • / search • Esc back", pageInfo)
		}
	case modeSearch:
		hints = fmt.Sprintf("Search: %s█  Enter/Esc confirm • Esc clear", m.searchQuery)
	case modeAdd:
		if m.activeField == 0 {
			hints = "Tab/Enter next field • Esc cancel • Ctrl+V paste"
		} else {
			hints = "Enter confirm • Esc cancel • Ctrl+V paste"
		}
	case modeEdit:
		hints = "Enter confirm • Esc cancel • Ctrl+V paste"
	case modeDelete:
		hints = "y confirm • any key cancel"
	}
	sb.WriteString(m.footer.View(hints))

	return tea.NewView(sb.String())
}
