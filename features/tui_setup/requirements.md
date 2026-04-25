# requirements.md — tui_setup
## User decisions
| Decision | Choice |
|----------|--------|
| Navigation | Main menu as a selectable list (arrow keys + Enter) |
| Configuration file | `ekvs-tui.yaml` in the current directory (default); override with `--config <path>` |
| File absent (default path) | Interactive wizard: asks for server URL and identity file, offers to save with custom name |
| File absent (`--config` explicit) | Fatal error with descriptive message, non-zero exit code |
| Malformed file | Fatal error with descriptive message |
| Default theme | Adaptive (`adaptive`, uses `lipgloss.AdaptiveColor`) |
| Available themes | `adaptive` (default), `hacker` (green on black) |
| Theme selection | `theme` field in YAML; unrecognised value → fatal error |
| Multiple profiles | The YAML file may contain a list of profiles, each with `name`, `server_url`, `identity_file`, `theme` |
| Profile selection | If the file contains one profile it is used directly; if it contains more than one, a profile selection screen is shown at startup |
| UI libraries | `charm.land/bubbletea/v2 v2.0.6`, `github.com/charmbracelet/bubbles v1.0.0`, `github.com/charmbracelet/lipgloss v1.1.0` |
| `--config` flag | Included in this phase; parsed with stdlib `flag` |
| Internal packages | `internal/tui/config`, `internal/tui/theme`, `internal/tui/app` |
---
## Scope
### In scope
- Add dependencies `charm.land/bubbletea/v2`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`.
- Package `internal/tui/config`: structs `Profile` and `ConfigFile`, functions `LoadFromFile` and `Save`, default-merge logic.
- Package `internal/tui/theme`: `Theme` interface, `AdaptiveTheme` and `HackerTheme` implementations, factory `NewTheme(name string) (Theme, error)`.
- Package `internal/tui/app`: main bubbletea model with a placeholder list menu, keyboard navigation.
- Profile selection screen (bubbletea) shown at startup when the file contains more than one profile.
- First-run wizard: interactive collection of `name`, `server_url` and `identity_file` via a custom text input, with an offer to save to a file.
- `cmd/tui/main.go`: `--config` flag, full bootstrap (load file → wizard if absent → profile selection if multiple → start bubbletea app).
- Unit tests for `internal/tui/config` and `internal/tui/theme`.
- Example file `ekvs-tui.yaml.example` in the repository root.
### Out of scope
- HTTP calls to the server (deferred to `tui_auth` and later).
- SSH authentication (deferred to `tui_auth`).
- Projects, Secrets, Settings screens (placeholder screens are sufficient).
- Hot-reload of the configuration.
- Semantic validation of `server_url` and `identity_file` (e.g. server reachability, file existence).
- Profile CRUD from within the running application (add/remove profiles at runtime).
---
## YAML structure — `ekvs-tui.yaml`
The file contains a list of profiles. Each profile has a unique `name` and its own connection parameters. The `theme` field is per-profile, allowing different themes for different servers.
```yaml
# ekvs-tui.yaml — EKVS TUI client configuration
profiles:
  - name:          "local"
    server_url:    "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "adaptive"
  - name:          "production"
    server_url:    "https://ekvs.example.com"
    identity_file: "~/.ssh/id_rsa"
    theme:         "hacker"
```
**Rules:**
- `name` must be non-empty and unique within the file; duplicate names cause a load error.
- If `theme` is omitted in a profile, the default `"adaptive"` is used.
- If `server_url` or `identity_file` are omitted, their respective defaults are used.
- A file with an empty `profiles` list is treated as absent (wizard is started).
---
## Package `internal/tui/config`
```go
package config
// Profile represents a single connection profile.
type Profile struct {
    Name         string `yaml:"name"`
    ServerURL    string `yaml:"server_url"`
    IdentityFile string `yaml:"identity_file"`
    Theme        string `yaml:"theme"`
}
// ConfigFile is the root structure of the YAML configuration file.
type ConfigFile struct {
    Profiles []Profile `yaml:"profiles"`
}
// DefaultProfile returns a Profile populated with default values.
func DefaultProfile() Profile
// LoadFromFile loads a ConfigFile from the YAML file at path.
//   - Returns (nil, nil) if required is false and the file does not exist,
//     or if the profiles list is empty.
//   - Returns an error if required is true and the file does not exist.
//   - Returns an error if the file contains invalid YAML.
//   - Returns an error if a profile has an empty name or two profiles share the same name.
func LoadFromFile(path string, required bool) (*ConfigFile, error)
// Save serialises cf to YAML and writes it to the file at path.
func Save(path string, cf *ConfigFile) error
```
**Defaults per `Profile`:**
| Field | Default |
|-------|---------|
| `ServerURL` | `"http://127.0.0.1:8080"` |
| `IdentityFile` | `"~/.ssh/id_ed25519"` |
| `Theme` | `"adaptive"` |
Fields omitted in a YAML profile are filled with `DefaultProfile()` values after unmarshal.
---
## Package `internal/tui/theme`
```go
package theme
import "github.com/charmbracelet/lipgloss"
// Theme groups all lipgloss styles used by the application.
type Theme interface {
    PrimaryColor() lipgloss.TerminalColor
    SecondaryColor() lipgloss.TerminalColor
    BackgroundColor() lipgloss.TerminalColor
    ErrorColor() lipgloss.TerminalColor
    TitleStyle() lipgloss.Style
    MenuItemStyle() lipgloss.Style
    SelectedMenuItemStyle() lipgloss.Style
    StatusBarStyle() lipgloss.Style
    ErrorStyle() lipgloss.Style
}
// NewTheme returns the Theme corresponding to the given name.
// Valid names: "adaptive", "hacker".
// Returns an error for unrecognised names.
func NewTheme(name string) (Theme, error)
```
**Themes:**
- `AdaptiveTheme`: uses `lipgloss.AdaptiveColor{Light: "...", Dark: "..."}` to adapt to the terminal.
- `HackerTheme`: green `#00FF41` on black background `#0D0208`, Matrix-inspired.
---
## Package `internal/tui/app`
```go
package app
import (
    tea "charm.land/bubbletea/v2"
    "ekvs/internal/tui/theme"
)
// MenuItem represents a main menu entry.
type MenuItem struct {
    Label string
    ID    string
}
// Model is the main bubbletea model of the application.
type Model struct { /* unexported fields */ }
// New creates a new Model with the given theme.
func New(t theme.Theme) Model
// Init, Update, View implement the tea.Model interface.
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() tea.View
```
**Placeholder menu items (in order):**
| ID | Label |
|----|-------|
| `projects` | `Projects` |
| `secrets` | `Secrets` |
| `settings` | `Settings` |
| `quit` | `Quit` |
**Keys:**
- `↑` / `k`: previous item
- `↓` / `j`: next item
- `Enter`: select item (`quit` exits; others are no-ops in this phase)
- `q` / `Ctrl+C`: exit the application
---
## `cmd/tui/main.go`
```go
// Supported flags:
//   --config <path>   path to the YAML configuration file (default: "ekvs-tui.yaml")
//
// Bootstrap logic:
//  1. Parse flags.
//  2. Determine whether the config path is the default or was explicitly provided.
//  3. Call config.LoadFromFile(path, explicit).
//     - (nil, nil)          → file absent or empty: start the first-run wizard.
//     - (nil, err)          → missing (explicit=true) or malformed: log.Fatal.
//     - (cf, nil) 1 profile → use that profile directly.
//     - (cf, nil) N profiles → show the profile selection screen.
//  4. Call theme.NewTheme(profile.Theme) → fatal error if unrecognised.
//  5. Create app.New(t) and run tea.NewProgram(model).Run().
```
**First-run wizard** (package `internal/tui/wizard`):
- Collects `name`, `server_url` and `identity_file` via a custom bubbletea v2 text input (pre-filled with defaults for URL and identity file).
- Asks: `Save configuration to file? [y/N]`.
- If yes, asks for the file name (default: `ekvs-tui.yaml`) and calls `config.Save`.
- Returns the collected `Profile`.
**Profile selection screen** (package `internal/tui/profileselect`):
- List of profiles showing `name` and `server_url` on each row.
- Navigation `↑↓/jk`, `Enter` to select, `q`/`Ctrl+C` to quit (returns an error).
- Returns the chosen `Profile`.
---
## Dependencies to add
```bash
go get charm.land/bubbletea/v2@v2.0.6
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/charmbracelet/lipgloss@v1.1.0
go mod tidy
```
> **Note:** `github.com/charmbracelet/bubbles v1.0.0` targets bubbletea v1 and is therefore
> **not used directly** in TUI models. All interactive components use `charm.land/bubbletea/v2`.
