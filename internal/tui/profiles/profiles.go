package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	tuiconfig "ekvs/internal/tui/config"
	"ekvs/internal/tui/theme"
)

// ── types ─────────────────────────────────────────────────────────────────────

// mode represents the current interaction mode of the Profiles screen.
type mode int

const (
	modeList          mode = iota
	modeCreate             // multi-step form for a new profile
	modeEdit               // multi-step form for an existing profile
	modeDeleteConfirm      // inline y/N prompt before deleting
	modeReloadPrompt       // y/N prompt after saving changes to the active profile
)

// formStep is the step within the create/edit form.
type formStep int

const (
	stepName formStep = iota
	stepServerURL
	stepIdentity
	stepTheme
)

// identityInputMode determines how the identity_file field is being filled.
type identityInputMode int

const (
	identityModePick   identityInputMode = iota // select from the discovered list
	identityModeManual                          // type a custom path
)

// supportedThemes is the ordered list of theme names the user may choose from.
var supportedThemes = []string{"adaptive", "hacker"}

// ── textInput ─────────────────────────────────────────────────────────────────

// textInput is a minimal single-line text field compatible with Bubble Tea v2.
type textInput struct {
	value       string
	placeholder string
}

func newTextInput(placeholder, value string) textInput {
	return textInput{placeholder: placeholder, value: value}
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

// ── profileForm ───────────────────────────────────────────────────────────────

// profileForm accumulates the state for the create/edit form.
type profileForm struct {
	step    formStep
	isEdit  bool
	oldName string // stable identifier used by UpdateProfile on save

	name      textInput
	serverURL textInput

	// identity file – two input modes
	identMode       identityInputMode
	discovered      []string
	discoveryCursor int
	identityManual  textInput

	// theme selection
	themeIndex int // index into supportedThemes
}

// selectedIdentity returns the current value of the identity_file field.
func (f profileForm) selectedIdentity() string {
	if f.identMode == identityModeManual || len(f.discovered) == 0 {
		return f.identityManual.value
	}
	if f.discoveryCursor < len(f.discovered) {
		return f.discovered[f.discoveryCursor]
	}
	return ""
}

// selectedTheme returns the theme name at themeIndex, defaulting to "adaptive".
func (f profileForm) selectedTheme() string {
	if f.themeIndex >= 0 && f.themeIndex < len(supportedThemes) {
		return supportedThemes[f.themeIndex]
	}
	return "adaptive"
}

// toProfile builds a Profile from the current form state.
// Blank optional fields fall back to DefaultProfile values.
func (f profileForm) toProfile() tuiconfig.Profile {
	def := tuiconfig.DefaultProfile()
	name := strings.TrimSpace(f.name.value)
	serverURL := strings.TrimSpace(f.serverURL.value)
	if serverURL == "" {
		serverURL = def.ServerURL
	}
	identity := strings.TrimSpace(f.selectedIdentity())
	if identity == "" {
		identity = def.IdentityFile
	}
	return tuiconfig.Profile{
		Name:         name,
		ServerURL:    serverURL,
		IdentityFile: identity,
		Theme:        f.selectedTheme(),
	}
}

// ── Model ─────────────────────────────────────────────────────────────────────

// Model is the Bubble Tea model for the Profiles screen.
type Model struct {
	theme         theme.Theme
	config        *tuiconfig.ConfigFile
	configPath    string
	activeProfile string // name of the currently active profile

	cursor int
	mode   mode

	form         profileForm
	deleteTarget string

	// modeReloadPrompt: the profile that was saved and awaits reload confirmation.
	pendingReload tuiconfig.Profile

	err string

	// sshDirFn is the function used to locate the SSH directory.
	// Defaults to tuiconfig.SSHDir; overridable via WithSSHDirFn for tests.
	sshDirFn func() (string, error)
}

// New creates the Profiles screen model.
//
//   - cfg is the current in-memory ConfigFile; must not be nil.
//   - configPath is the file path used for persistence.
//   - activeProfileName is the Name of the profile currently loaded in root.
//   - t is the active theme.
func New(cfg *tuiconfig.ConfigFile, configPath string, activeProfileName string, t theme.Theme) Model {
	if cfg == nil {
		cfg = &tuiconfig.ConfigFile{}
	}
	return Model{
		theme:         t,
		config:        cfg,
		configPath:    configPath,
		activeProfile: activeProfileName,
	}
}

// WithSSHDirFn replaces the SSH directory discovery function.
// Intended for testing; the returned Model is a shallow copy with the function overridden.
func (m Model) WithSSHDirFn(fn func() (string, error)) Model {
	m.sshDirFn = fn
	return m
}

// ── Init / UpdateTyped / Update ───────────────────────────────────────────────

// Init implements tea.Model. The profiles screen has no async initialisation.
func (m Model) Init() tea.Cmd { return nil }

// UpdateTyped returns the concrete Model type directly, avoiding silent
// type-assertion failures in the root model.
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
	case tea.KeyPressMsg:
		m.err = "" // clear previous error on any key
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeCreate, modeEdit:
			return m.updateForm(msg)
		case modeDeleteConfirm:
			return m.updateDeleteConfirm(msg)
		case modeReloadPrompt:
			return m.updateReloadPrompt(msg)
		}
	}
	return m, nil
}

// ── list mode ─────────────────────────────────────────────────────────────────

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	profiles := m.config.Profiles
	switch msg.String() {
	case "up", "k":
		if len(profiles) > 0 {
			m.cursor = (m.cursor - 1 + len(profiles)) % len(profiles)
		}
	case "down", "j":
		if len(profiles) > 0 {
			m.cursor = (m.cursor + 1) % len(profiles)
		}
	case "enter", "s":
		if len(profiles) > 0 && m.cursor < len(profiles) {
			target := profiles[m.cursor]
			if target.Name != m.activeProfile {
				return m, func() tea.Msg { return SwitchMsg{Profile: target} }
			}
		}
	case "c":
		m.mode = modeCreate
		m.form = m.newCreateForm()
	case "e":
		if len(profiles) > 0 && m.cursor < len(profiles) {
			m.mode = modeEdit
			m.form = m.newEditForm(profiles[m.cursor])
		}
	case "d":
		if len(profiles) > 0 && m.cursor < len(profiles) {
			m.deleteTarget = profiles[m.cursor].Name
			m.mode = modeDeleteConfirm
		}
	case "esc":
		return m, func() tea.Msg { return BackMsg{} }
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// ── form mode ─────────────────────────────────────────────────────────────────

func (m Model) updateForm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.form.step {
	case stepName:
		return m.updateFormName(msg)
	case stepServerURL:
		return m.updateFormServerURL(msg)
	case stepIdentity:
		return m.updateFormIdentity(msg)
	case stepTheme:
		return m.updateFormTheme(msg)
	}
	return m, nil
}

func (m Model) updateFormName(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		// Cancel the form and return to list mode.
		m.mode = modeList
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.form.name.value)
		if val == "" {
			m.err = "profile name cannot be empty"
			return m, nil
		}
		// Check uniqueness; when editing allow keeping the same name.
		if _, _, exists := m.config.FindProfile(val); exists {
			if !(m.form.isEdit && val == m.form.oldName) {
				m.err = fmt.Sprintf("profile name %q already exists", val)
				return m, nil
			}
		}
		m.err = ""
		m.form.step = stepServerURL
	default:
		m.form.name = m.form.name.update(msg)
	}
	return m, nil
}

func (m Model) updateFormServerURL(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.form.step = stepName
	case "enter":
		m.form.step = stepIdentity
	default:
		m.form.serverURL = m.form.serverURL.update(msg)
	}
	return m, nil
}

func (m Model) updateFormIdentity(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.form.identMode == identityModeManual && len(m.form.discovered) > 0 {
			// Go back to the pick list instead of going to the previous step.
			m.form.identMode = identityModePick
			return m, nil
		}
		m.form.step = stepServerURL
	case "m":
		if m.form.identMode == identityModePick {
			m.form.identMode = identityModeManual
			return m, nil
		}
		// In manual mode 'm' is a regular character.
		m.form.identityManual = m.form.identityManual.update(msg)
	case "up":
		if m.form.identMode == identityModePick && len(m.form.discovered) > 0 {
			n := len(m.form.discovered)
			m.form.discoveryCursor = (m.form.discoveryCursor - 1 + n) % n
		}
		// "up" carries no text; ignore in manual mode.
	case "k":
		if m.form.identMode == identityModePick && len(m.form.discovered) > 0 {
			n := len(m.form.discovered)
			m.form.discoveryCursor = (m.form.discoveryCursor - 1 + n) % n
		} else if m.form.identMode == identityModeManual {
			m.form.identityManual = m.form.identityManual.update(msg)
		}
	case "down":
		if m.form.identMode == identityModePick && len(m.form.discovered) > 0 {
			n := len(m.form.discovered)
			m.form.discoveryCursor = (m.form.discoveryCursor + 1) % n
		}
		// "down" carries no text; ignore in manual mode.
	case "j":
		if m.form.identMode == identityModePick && len(m.form.discovered) > 0 {
			n := len(m.form.discovered)
			m.form.discoveryCursor = (m.form.discoveryCursor + 1) % n
		} else if m.form.identMode == identityModeManual {
			m.form.identityManual = m.form.identityManual.update(msg)
		}
	case "enter":
		m.form.step = stepTheme
	default:
		if m.form.identMode == identityModeManual {
			m.form.identityManual = m.form.identityManual.update(msg)
		}
	}
	return m, nil
}

func (m Model) updateFormTheme(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.form.step = stepIdentity
	case "up", "k":
		n := len(supportedThemes)
		m.form.themeIndex = (m.form.themeIndex - 1 + n) % n
	case "down", "j":
		n := len(supportedThemes)
		m.form.themeIndex = (m.form.themeIndex + 1) % n
	case "enter":
		return m.saveForm()
	}
	return m, nil
}

// saveForm persists the form result and emits the appropriate message.
// On save failure the in-memory config is restored to its pre-mutation state.
func (m Model) saveForm() (tea.Model, tea.Cmd) {
	// Snapshot the current slice so we can roll back on disk-write failure.
	backup := make([]tuiconfig.Profile, len(m.config.Profiles))
	copy(backup, m.config.Profiles)

	profile := m.form.toProfile()

	var mutErr error
	if m.form.isEdit {
		mutErr = m.config.UpdateProfile(m.form.oldName, profile)
	} else {
		mutErr = m.config.UpsertProfile(profile)
	}
	if mutErr != nil {
		m.err = mutErr.Error()
		return m, nil
	}

	if saveErr := tuiconfig.Save(m.configPath, m.config); saveErr != nil {
		m.config.Profiles = backup // restore in-memory state
		m.err = fmt.Sprintf("save failed: %v", saveErr)
		return m, nil
	}

	cfg := m.config
	isActiveEdit := m.form.isEdit && m.form.oldName == m.activeProfile

	if isActiveEdit {
		// Update the tracked active profile name to handle renames correctly.
		m.activeProfile = profile.Name
		m.pendingReload = profile
		m.mode = modeReloadPrompt
		// Do NOT emit ConfigChangedMsg here: the reload prompt is still pending.
		// Root will receive the message only after the user answers the prompt.
		return m, nil
	}

	// Non-active edit or create: return to list and notify root.
	m.mode = modeList
	if !m.form.isEdit {
		// Position cursor at the newly created profile.
		for i, p := range m.config.Profiles {
			if p.Name == profile.Name {
				m.cursor = i
				break
			}
		}
	}
	return m, func() tea.Msg { return ConfigChangedMsg{Config: cfg} }
}

// ── delete confirm mode ───────────────────────────────────────────────────────

func (m Model) updateDeleteConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.doDelete()
	case "n", "N", "esc":
		m.mode = modeList
		m.deleteTarget = ""
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) doDelete() (tea.Model, tea.Cmd) {
	// Snapshot for rollback.
	backup := make([]tuiconfig.Profile, len(m.config.Profiles))
	copy(backup, m.config.Profiles)

	target := m.deleteTarget
	isActive := target == m.activeProfile

	if err := m.config.DeleteProfile(target); err != nil {
		m.err = err.Error()
		m.mode = modeList
		return m, nil
	}

	if err := tuiconfig.Save(m.configPath, m.config); err != nil {
		m.config.Profiles = backup
		m.err = fmt.Sprintf("save failed: %v", err)
		m.mode = modeList
		return m, nil
	}

	// Clamp cursor after deletion.
	if m.cursor >= len(m.config.Profiles) && m.cursor > 0 {
		m.cursor = len(m.config.Profiles) - 1
	}

	m.mode = modeList
	m.deleteTarget = ""
	cfg := m.config
	return m, func() tea.Msg {
		return ConfigChangedMsg{Config: cfg, ActiveDeleted: isActive}
	}
}

// ── reload prompt mode ────────────────────────────────────────────────────────

func (m Model) updateReloadPrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		profile := m.pendingReload
		cfg := m.config
		m.mode = modeList
		m.pendingReload = tuiconfig.Profile{}
		return m, func() tea.Msg { return ReloadActiveMsg{Profile: profile, Config: cfg} }
	case "n", "N", "enter", "esc":
		cfg := m.config
		m.mode = modeList
		m.pendingReload = tuiconfig.Profile{}
		// Notify root to update its in-memory config even though the active
		// profile is NOT being reloaded into the session.
		return m, func() tea.Msg { return ConfigChangedMsg{Config: cfg} }
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// ── form constructors ─────────────────────────────────────────────────────────

func (m Model) newCreateForm() profileForm {
	def := tuiconfig.DefaultProfile()
	discovered := m.discoverSSHKeys()
	identMode := identityModePick
	if len(discovered) == 0 {
		identMode = identityModeManual
	}
	themeIdx := 0
	for i, name := range supportedThemes {
		if name == def.Theme {
			themeIdx = i
			break
		}
	}
	return profileForm{
		step:       stepName,
		isEdit:     false,
		name:       newTextInput("e.g. production", ""),
		serverURL:  newTextInput("", def.ServerURL),
		identMode:  identMode,
		discovered: discovered,
		// Empty value so the user types a fresh path; placeholder shows the default.
		identityManual: newTextInput(def.IdentityFile, ""),
		themeIndex:     themeIdx,
	}
}

func (m Model) newEditForm(p tuiconfig.Profile) profileForm {
	discovered := m.discoverSSHKeys()
	identMode := identityModeManual
	discoveryCursor := 0
	for i, d := range discovered {
		if d == p.IdentityFile {
			identMode = identityModePick
			discoveryCursor = i
			break
		}
	}
	if len(discovered) == 0 {
		identMode = identityModeManual
	}
	themeIdx := 0
	for i, name := range supportedThemes {
		if name == p.Theme {
			themeIdx = i
			break
		}
	}
	return profileForm{
		step:            stepName,
		isEdit:          true,
		oldName:         p.Name,
		name:            newTextInput("", p.Name),
		serverURL:       newTextInput("", p.ServerURL),
		identMode:       identMode,
		discovered:      discovered,
		discoveryCursor: discoveryCursor,
		identityManual:  newTextInput("", p.IdentityFile),
		themeIndex:      themeIdx,
	}
}

// discoverSSHKeys returns paths of likely SSH private-key files in the SSH
// directory. Uses m.sshDirFn (defaults to tuiconfig.SSHDir) so tests can
// override the directory without touching the real ~/.ssh.
func (m Model) discoverSSHKeys() []string {
	fn := m.sshDirFn
	if fn == nil {
		fn = tuiconfig.SSHDir
	}
	sshDir, err := fn()
	if err != nil || sshDir == "" {
		return nil
	}
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return nil
	}
	skip := map[string]bool{
		"known_hosts":     true,
		"known_hosts.old": true,
		"config":          true,
		"authorized_keys": true,
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pub") || skip[name] {
			continue
		}
		keys = append(keys, filepath.Join(sshDir, name))
	}
	return keys
}

// ── View ─────────────────────────────────────────────────────────────────────

// View implements tea.Model.
func (m Model) View() tea.View {
	var sb strings.Builder
	t := m.theme

	switch m.mode {
	case modeList:
		sb.WriteString(t.TitleStyle().Render("Profiles"))
		sb.WriteString("\n")
		if len(m.config.Profiles) == 0 {
			sb.WriteString(t.MenuItemStyle().Render("  (no profiles)"))
			sb.WriteString("\n")
		} else {
			for i, p := range m.config.Profiles {
				active := ""
				if p.Name == m.activeProfile {
					active = " *"
				}
				line := fmt.Sprintf("%s%s  %s  %s  %s",
					p.Name, active, p.ServerURL, p.IdentityFile, p.Theme)
				if i == m.cursor {
					sb.WriteString(t.SelectedMenuItemStyle().Render("> " + line))
				} else {
					sb.WriteString(t.MenuItemStyle().Render("  " + line))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
		if m.err != "" {
			sb.WriteString(t.ErrorStyle().Render(m.err))
			sb.WriteString("\n")
		}
		sb.WriteString(t.StatusBarStyle().Render(
			"↑/↓ navigate • Enter/s switch • c create • e edit • d delete • Esc back • q quit"))

	case modeCreate, modeEdit:
		title := "Create Profile"
		if m.mode == modeEdit {
			title = "Edit Profile"
		}
		sb.WriteString(t.TitleStyle().Render(title))
		sb.WriteString("\n\n")
		m.renderForm(&sb)

	case modeDeleteConfirm:
		sb.WriteString(t.TitleStyle().Render("Delete Profile"))
		sb.WriteString("\n\n")
		sb.WriteString(t.ErrorStyle().Render(
			fmt.Sprintf("Delete profile %q? [y/N]", m.deleteTarget)))
		sb.WriteString("\n\n")
		sb.WriteString(t.StatusBarStyle().Render("y confirm • n/Esc cancel"))

	case modeReloadPrompt:
		sb.WriteString(t.TitleStyle().Render("Profile Updated"))
		sb.WriteString("\n\n")
		sb.WriteString(t.MenuItemStyle().Render("Changes saved. Reload profile now? [y/N]"))
		sb.WriteString("\n\n")
		sb.WriteString(t.StatusBarStyle().Render("y reload • n/Esc keep current session"))
	}

	return tea.NewView(sb.String())
}

// renderForm writes the form view into sb.
func (m Model) renderForm(sb *strings.Builder) {
	t := m.theme
	f := m.form

	// Summary of already-confirmed fields.
	if f.step > stepName {
		sb.WriteString(fmt.Sprintf("Name:   %s\n", f.name.value))
	}
	if f.step > stepServerURL {
		sb.WriteString(fmt.Sprintf("Server: %s\n", f.serverURL.value))
	}
	if f.step > stepIdentity {
		sb.WriteString(fmt.Sprintf("Key:    %s\n", f.selectedIdentity()))
	}

	switch f.step {
	case stepName:
		sb.WriteString("Profile name:\n")
		sb.WriteString("  " + f.name.view(true) + "\n")

	case stepServerURL:
		sb.WriteString("\nServer URL:\n")
		sb.WriteString("  " + f.serverURL.view(true) + "\n")

	case stepIdentity:
		sb.WriteString("\nSSH identity file:\n")
		if f.identMode == identityModePick {
			if len(f.discovered) == 0 {
				sb.WriteString(t.MenuItemStyle().Render("  (no keys discovered — press m to enter path manually)"))
				sb.WriteString("\n")
			} else {
				for i, k := range f.discovered {
					if i == f.discoveryCursor {
						sb.WriteString(t.SelectedMenuItemStyle().Render("> " + k))
					} else {
						sb.WriteString(t.MenuItemStyle().Render("  " + k))
					}
					sb.WriteString("\n")
				}
				sb.WriteString(t.MenuItemStyle().Render("  m — enter a custom path"))
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("  " + f.identityManual.view(true) + "\n")
		}

	case stepTheme:
		sb.WriteString("\nTheme:\n")
		for i, name := range supportedThemes {
			if i == f.themeIndex {
				sb.WriteString(t.SelectedMenuItemStyle().Render("> " + name))
			} else {
				sb.WriteString(t.MenuItemStyle().Render("  " + name))
			}
			sb.WriteString("\n")
		}
	}

	if m.err != "" {
		sb.WriteString("\n")
		sb.WriteString(t.ErrorStyle().Render(m.err))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	switch f.step {
	case stepName, stepServerURL:
		sb.WriteString(t.StatusBarStyle().Render("Enter confirm • Esc back • Ctrl+C quit"))
	case stepIdentity:
		if f.identMode == identityModePick {
			sb.WriteString(t.StatusBarStyle().Render("↑/↓ select key • m manual path • Enter confirm • Esc back"))
		} else {
			sb.WriteString(t.StatusBarStyle().Render("Enter confirm • Esc back • Ctrl+C quit"))
		}
	case stepTheme:
		sb.WriteString(t.StatusBarStyle().Render("↑/↓ select theme • Enter confirm • Esc back"))
	}
}
