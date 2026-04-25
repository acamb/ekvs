# plan.md — tui_setup
## Ordered task list
---
### 1. Add UI dependencies
```bash
go get charm.land/bubbletea/v2@v2.0.6
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/charmbracelet/lipgloss@v1.1.0
go mod tidy
```
Verify that `go.mod` contains the three direct dependencies.
---
### 2. Implement `internal/tui/config`
Create `internal/tui/config/config.go`.
```go
package config
import (
    "errors"
    "fmt"
    "os"
    "gopkg.in/yaml.v3"
)
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
func DefaultProfile() Profile {
    return Profile{
        ServerURL:    "http://127.0.0.1:8080",
        IdentityFile: "~/.ssh/id_ed25519",
        Theme:        "adaptive",
    }
}
// LoadFromFile loads a ConfigFile from the YAML file at path.
// Returns (nil, nil) if the file does not exist (required=false) or the list is empty.
// Returns an error if required=true and the file does not exist, the YAML is malformed,
// or two profiles share the same name.
func LoadFromFile(path string, required bool) (*ConfigFile, error) { ... }
// Save serialises cf to YAML and writes it to path.
func Save(path string, cf *ConfigFile) error { ... }
```
Default-merge logic after unmarshal: for each `Profile` in the list, empty string fields
are replaced with the corresponding value from `DefaultProfile()`.
Duplicate-name validation:
```go
seen := map[string]bool{}
for _, p := range cf.Profiles {
    if p.Name == "" {
        return nil, fmt.Errorf("profile at position %d has no name", i+1)
    }
    if seen[p.Name] {
        return nil, fmt.Errorf("duplicate profile name: %q", p.Name)
    }
    seen[p.Name] = true
}
```
---
### 3. Write unit tests for `internal/tui/config`
Create `internal/tui/config/config_test.go` with table-driven tests for `LoadFromFile` and `Save`:
| Scenario | Expected behaviour |
|----------|--------------------|
| Single complete profile | Returns ConfigFile with that profile |
| Multiple complete profiles | Returns ConfigFile with all profiles |
| Partial profile | Missing fields use `DefaultProfile()` values |
| File absent + `required=false` | Returns `(nil, nil)` |
| File absent + `required=true` | Returns error |
| Malformed YAML | Returns error |
| Empty `profiles` list | Returns `(nil, nil)` |
| Two profiles with the same `name` | Returns error |
| Profile with empty `name` | Returns error |
| `Save` + `LoadFromFile` round-trip | Saved and reloaded ConfigFile is identical to the original |
---
### 4. Implement `internal/tui/theme`
Create `internal/tui/theme/theme.go`.
```go
package theme
import "github.com/charmbracelet/lipgloss"
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
// AdaptiveTheme adapts to the terminal's light/dark colour scheme.
type AdaptiveTheme struct{}
// HackerTheme is a green-on-black Matrix-inspired theme.
type HackerTheme struct{}
// NewTheme returns the theme corresponding to the given name.
// Valid values: "adaptive", "hacker".
func NewTheme(name string) (Theme, error) {
    switch name {
    case "adaptive":
        return AdaptiveTheme{}, nil
    case "hacker":
        return HackerTheme{}, nil
    default:
        return nil, fmt.Errorf("unknown theme: %q (valid values: adaptive, hacker)", name)
    }
}
```
**`HackerTheme` palette** (constants defined in the package):
```go
const (
    hackerGreen      = lipgloss.Color("#00FF41")
    hackerGreenDim   = lipgloss.Color("#008F11")
    hackerBackground = lipgloss.Color("#0D0208")
    hackerRed        = lipgloss.Color("#FF0000")
)
```
**`AdaptiveTheme` palette** uses `lipgloss.AdaptiveColor`:
```go
var adaptivePrimary = lipgloss.AdaptiveColor{Light: "#5A5A5A", Dark: "#DDDDDD"}
// etc.
```
---
### 5. Write unit tests for `internal/tui/theme`
Create `internal/tui/theme/theme_test.go` with table-driven tests for `NewTheme`:
| Input | Expected result |
|-------|-----------------|
| `"adaptive"` | No error, `AdaptiveTheme` instance |
| `"hacker"` | No error, `HackerTheme` instance |
| `""` | Error |
| `"unknown"` | Error |
Additional smoke tests: call every method on both themes and verify no panics / nil returns.
---
### 6. Implement `internal/tui/app`
Create `internal/tui/app/app.go`.
```go
package app
import (
    tea "charm.land/bubbletea/v2"
    "ekvs/internal/tui/theme"
)
type MenuItem struct {
    Label string
    ID    string
}
var defaultMenuItems = []MenuItem{
    {ID: "projects", Label: "Projects"},
    {ID: "secrets",  Label: "Secrets"},
    {ID: "settings", Label: "Settings"},
    {ID: "quit",     Label: "Quit"},
}
type Model struct {
    items    []MenuItem
    cursor   int
    theme    theme.Theme
    quitting bool
}
func New(t theme.Theme) Model {
    return Model{items: defaultMenuItems, theme: t}
}
```
`Update` logic:
- `"up"` / `"k"`: decrement `cursor` (with wrap-around).
- `"down"` / `"j"`: increment `cursor` (with wrap-around).
- `"enter"`: if selected item has `ID == "quit"` → `tea.Quit`; otherwise no-op (placeholder for future phases).
- `"q"` / `"ctrl+c"`: `tea.Quit`.
`View` logic:
- Title rendered with `theme.TitleStyle()`.
- Item list: each item uses `theme.MenuItemStyle()`; the current item uses `theme.SelectedMenuItemStyle()` with a visual cursor (`"> "`).
- Footer with shortcuts rendered with `theme.StatusBarStyle()`.
---
### 7. Implement the first-run wizard
Create `internal/tui/wizard/wizard.go`.
The wizard is a self-contained bubbletea program with sequential steps:
1. **Step 1** — `name` input (empty, placeholder `"e.g. local"`).
2. **Step 2** — `server_url` input (pre-filled with default).
3. **Step 3** — `identity_file` input (pre-filled with default).
4. **Step 4** — Save confirmation: `y` for yes, `n`/`Enter` for no.
5. **Step 5** (conditional) — File name input (pre-filled with `ekvs-tui.yaml`).
> **Note:** `bubbles/textinput` targets bubbletea v1 and is incompatible with v2.
> A minimal custom `textInput` struct is used instead, handling `KeyPressMsg` and `Key.Text`.
```go
package wizard
import (
    "ekvs/internal/tui/config"
    "ekvs/internal/tui/theme"
)
// Run starts the interactive wizard and returns the collected profile.
// If the user chooses to save, it writes the YAML file with the created profile.
func Run(t theme.Theme) (config.Profile, error)
```
---
### 8. Implement the profile selection screen
Create `internal/tui/profileselect/profileselect.go`.
```go
package profileselect
import (
    "ekvs/internal/tui/config"
    "ekvs/internal/tui/theme"
)
// Run displays the profile list and returns the profile chosen by the user.
// Each row shows: "  <name>  (<server_url>)".
// Navigation: ↑/↓/j/k, Enter to select, q/Ctrl+C to quit (returns error).
func Run(profiles []config.Profile, t theme.Theme) (config.Profile, error)
```
---
### 9. Update `cmd/tui/main.go`
Replace the stub with the full bootstrap:
```go
func main() {
    const defaultConfig = "ekvs-tui.yaml"
    configPath := flag.String("config", defaultConfig, "path to the YAML configuration file")
    flag.Parse()
    explicit := *configPath != defaultConfig
    cf, err := tuiconfig.LoadFromFile(*configPath, explicit)
    if err != nil {
        log.Fatalf("error loading configuration: %v", err)
    }
    defaultTheme, _ := theme.NewTheme("adaptive")
    var profile tuiconfig.Profile
    if cf == nil {
        profile, err = wizard.Run(defaultTheme)
        if err != nil {
            log.Fatalf("wizard interrupted: %v", err)
        }
    } else if len(cf.Profiles) == 1 {
        profile = cf.Profiles[0]
    } else {
        profile, err = profileselect.Run(cf.Profiles, defaultTheme)
        if err != nil {
            log.Fatalf("profile selection interrupted: %v", err)
        }
    }
    t, err := theme.NewTheme(profile.Theme)
    if err != nil {
        log.Fatalf("invalid theme configuration: %v", err)
    }
    m := app.New(t)
    p := tea.NewProgram(m)
    if _, err := p.Run(); err != nil {
        log.Printf("TUI error: %v", err)
        os.Exit(1)
    }
}
```
---
### 10. Create `ekvs-tui.yaml.example`
Create the example file in the repository root:
```yaml
# ekvs-tui.yaml.example — EKVS TUI client configuration example
# Rename to ekvs-tui.yaml or specify the path with --config <path>
profiles:
  - name:          "local"
    server_url:    "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "adaptive"   # values: adaptive | hacker
  - name:          "production"
    server_url:    "https://ekvs.example.com"
    identity_file: "~/.ssh/id_rsa"
    theme:         "hacker"
```
---
### 11. `go mod tidy` and build verification
```bash
go mod tidy
go build ./cmd/tui/...
go test ./internal/tui/...
```
All commands must complete without errors.
