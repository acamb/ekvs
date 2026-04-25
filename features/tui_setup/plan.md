# plan.md — tui_setup

## Lista ordinata dei task

---

### 1. Aggiungere le dipendenze UI

```bash
go get charm.land/bubbletea/v2@v2.0.6
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/charmbracelet/lipgloss@v1.1.0
go mod tidy
```

Verificare che `go.mod` contenga le tre dipendenze dirette.

---

### 2. Implementare `internal/tui/config`

Creare il file `internal/tui/config/config.go`.

Struttura del package:

```go
package config

import (
    "fmt"
    "os"
    "gopkg.in/yaml.v3"
)

// Profile rappresenta un singolo profilo di connessione.
type Profile struct {
    Name         string `yaml:"name"`
    ServerURL    string `yaml:"server_url"`
    IdentityFile string `yaml:"identity_file"`
    Theme        string `yaml:"theme"`
}

// ConfigFile è la struttura radice del file YAML.
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

// LoadFromFile carica il ConfigFile dal file YAML a path.
// Restituisce (nil, nil) se il file non esiste (required=false) o se la lista è vuota.
// Restituisce errore se required=true e il file non esiste, o se il YAML è malformato,
// o se due profili hanno lo stesso name.
func LoadFromFile(path string, required bool) (*ConfigFile, error) { ... }

// Save serializza cf in YAML e scrive il file a path.
func Save(path string, cf *ConfigFile) error { ... }
```

Logica di merge dei default dopo l'unmarshal: per ogni `Profile` nella lista, se un campo stringa è vuoto (`""`), viene sostituito con il corrispondente valore di `DefaultProfile()`.

Validazione nomi duplicati:
```go
seen := map[string]bool{}
for _, p := range cf.Profiles {
    if p.Name == "" {
        return nil, fmt.Errorf("profilo senza nome trovato")
    }
    if seen[p.Name] {
        return nil, fmt.Errorf("nome profilo duplicato: %q", p.Name)
    }
    seen[p.Name] = true
}
```

---

### 3. Scrivere unit test per `internal/tui/config`

Creare `internal/tui/config/config_test.go` con test table-driven per `LoadFromFile` e `Save`:

| Scenario | Comportamento atteso |
|----------|----------------------|
| File con un profilo completo | Restituisce ConfigFile con quel profilo |
| File con più profili completi | Restituisce ConfigFile con tutti i profili |
| Profilo con campi parziali | Campi mancanti usano i default di `DefaultProfile()` |
| File non esiste + `required=false` | Restituisce `(nil, nil)` |
| File non esiste + `required=true` | Restituisce errore |
| File YAML malformato | Restituisce errore |
| Lista `profiles` vuota | Restituisce `(nil, nil)` |
| Due profili con lo stesso `name` | Restituisce errore |
| Profilo con `name` vuoto | Restituisce errore |
| `Save` + `LoadFromFile` round-trip | ConfigFile salvato e ricaricato è identico all'originale |

---

### 4. Implementare `internal/tui/theme`

Creare `internal/tui/theme/theme.go`.

```go
package theme

import "github.com/charmbracelet/lipgloss"

type Theme interface {
    PrimaryColor() lipgloss.Color
    SecondaryColor() lipgloss.Color
    BackgroundColor() lipgloss.Color
    ErrorColor() lipgloss.Color
    TitleStyle() lipgloss.Style
    MenuItemStyle() lipgloss.Style
    SelectedMenuItemStyle() lipgloss.Style
    StatusBarStyle() lipgloss.Style
    ErrorStyle() lipgloss.Style
}

// AdaptiveTheme: si adatta a terminali chiari e scuri
type AdaptiveTheme struct{}

// HackerTheme: verde su nero, stile Matrix
type HackerTheme struct{}

// NewTheme restituisce il tema corrispondente al nome.
// Valori validi: "adaptive", "hacker".
func NewTheme(name string) (Theme, error) {
    switch name {
    case "adaptive":
        return AdaptiveTheme{}, nil
    case "hacker":
        return HackerTheme{}, nil
    default:
        return nil, fmt.Errorf("tema non riconosciuto: %q (valori validi: adaptive, hacker)", name)
    }
}
```

**Palette `HackerTheme`** (costanti da definire nel package):
```go
const (
    hackerGreen      = lipgloss.Color("#00FF41")
    hackerGreenDim   = lipgloss.Color("#008F11")
    hackerBackground = lipgloss.Color("#0D0208")
    hackerRed        = lipgloss.Color("#FF0000")
)
```

**Palette `AdaptiveTheme`** usa `lipgloss.AdaptiveColor`:
```go
var adaptivePrimary = lipgloss.AdaptiveColor{Light: "#5A5A5A", Dark: "#DDDDDD"}
// ecc.
```

---

### 5. Scrivere unit test per `internal/tui/theme`

Creare `internal/tui/theme/theme_test.go` con test table-driven per `NewTheme`:

| Input | Risultato atteso |
|-------|-----------------|
| `"adaptive"` | Nessun errore, istanza `AdaptiveTheme` |
| `"hacker"` | Nessun errore, istanza `HackerTheme` |
| `""` | Errore |
| `"unknown"` | Errore |

Test aggiuntivi: verificare che tutti i metodi dell'interfaccia `Theme` siano implementati per entrambi i temi (compilazione sufficiente, ma aggiungere smoke test che chiamano ogni metodo e verificano che il risultato non sia zero value).

---

### 6. Implementare `internal/tui/app`

Creare `internal/tui/app/app.go`.

```go
package app

import (
    "fmt"
    tea "charm.land/bubbletea/v2"
    "github.com/charmbracelet/lipgloss"
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

Logica `Update`:
- `tea.KeyMsg` con `"up"` / `"k"`: decrementa `cursor` (con wrap-around).
- `tea.KeyMsg` con `"down"` / `"j"`: incrementa `cursor` (con wrap-around).
- `tea.KeyMsg` con `"enter"`: se la voce selezionata ha `ID == "quit"` → `tea.Quit`; altrimenti, per ora non fa nulla (placeholder per fasi successive).
- `tea.KeyMsg` con `"q"` / `"ctrl+c"`: `tea.Quit`.

Logica `View`:
- Titolo stilizzato con `theme.TitleStyle()`.
- Lista voci: ogni voce usa `theme.MenuItemStyle()`, la voce corrente usa `theme.SelectedMenuItemStyle()` con un cursore visivo (es. `"> "`).
- Footer con shortcuts (`↑/↓ naviga • Enter seleziona • q esci`) stilizzato con `theme.StatusBarStyle()`.

---

### 7. Implementare il wizard di primo avvio

Creare `internal/tui/wizard/wizard.go`.

Il wizard è un programma bubbletea separato composto da step sequenziali:

1. **Step 1** — Input `name` (textinput vuoto, es. placeholder `"es. locale"`).
2. **Step 2** — Input `server_url` (textinput pre-compilato con il default).
3. **Step 3** — Input `identity_file` (textinput pre-compilato con il default).
4. **Step 4** — Conferma salvataggio: tasto `s` per sì, `n`/`Invio` per no.
5. **Step 5** (condizionale) — Input nome file (textinput pre-compilato con `ekvs-tui.yaml`).

```go
package wizard

import (
    "ekvs/internal/tui/config"
    "ekvs/internal/tui/theme"
)

// Run avvia il wizard interattivo e restituisce il profilo raccolto.
// Se l'utente sceglie di salvare, scrive il file con il profilo creato.
func Run(t theme.Theme) (config.Profile, error)
```

---

### 8. Implementare la schermata di selezione profilo

Creare `internal/tui/profileselect/profileselect.go`.

```go
package profileselect

import (
    "ekvs/internal/tui/config"
    "ekvs/internal/tui/theme"
)

// Run mostra la lista dei profili e restituisce quello scelto dall'utente.
// Ogni riga mostra: "  <name>  (<server_url>)".
// Navigazione: ↑/↓/j/k, Enter per selezionare, q/Ctrl+C per uscire (restituisce errore).
func Run(profiles []config.Profile, t theme.Theme) (config.Profile, error)
```

Usa `bubbles/list` oppure una lista custom analoga a `internal/tui/app` (preferire la soluzione più semplice e coerente con lo stile già scelto per il menu principale).

---

### 9. Aggiornare `cmd/tui/main.go`

Sostituire lo stub con il bootstrap completo:

```go
package main

import (
    "flag"
    "log"
    "os"

    tea "charm.land/bubbletea/v2"
    tuiconfig "ekvs/internal/tui/config"
    "ekvs/internal/tui/app"
    "ekvs/internal/tui/profileselect"
    "ekvs/internal/tui/theme"
    "ekvs/internal/tui/wizard"
)

func main() {
    defaultConfig := "ekvs-tui.yaml"
    configPath := flag.String("config", defaultConfig, "percorso al file di configurazione YAML")
    flag.Parse()

    explicit := *configPath != defaultConfig

    cf, err := tuiconfig.LoadFromFile(*configPath, explicit)
    if err != nil {
        log.Fatalf("errore nel caricamento della configurazione: %v", err)
    }

    // Tema di default per wizard e selezione profilo (prima che il profilo sia noto)
    defaultTheme, _ := theme.NewTheme("adaptive")

    var profile tuiconfig.Profile
    if cf == nil {
        // File assente o lista vuota: wizard di primo avvio
        profile, err = wizard.Run(defaultTheme)
        if err != nil {
            log.Fatalf("wizard interrotto: %v", err)
        }
    } else if len(cf.Profiles) == 1 {
        profile = cf.Profiles[0]
    } else {
        // Più profili: mostra la schermata di selezione
        profile, err = profileselect.Run(cf.Profiles, defaultTheme)
        if err != nil {
            log.Fatalf("selezione profilo interrotta: %v", err)
        }
    }

    t, err := theme.NewTheme(profile.Theme)
    if err != nil {
        log.Fatalf("configurazione tema non valida: %v", err)
    }

    model := app.New(t)
    p := tea.NewProgram(model)
    if _, err := p.Run(); err != nil {
        log.Fatalf("errore nell'esecuzione del TUI: %v", err)
        os.Exit(1)
    }
}
```

---

### 10. Creare `ekvs-tui.yaml.example`

Creare il file di esempio nella root del repository:

```yaml
# ekvs-tui.yaml.example — configurazione di esempio del client TUI EKVS
# Rinominare in ekvs-tui.yaml o specificare il path con --config

profiles:
  - name:          "locale"
    server_url:    "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "adaptive"   # valori: adaptive | hacker

  - name:          "produzione"
    server_url:    "https://ekvs.example.com"
    identity_file: "~/.ssh/id_rsa"
    theme:         "hacker"
```

---

### 11. `go mod tidy` e verifica compilazione

```bash
go mod tidy
go build ./cmd/tui/...
go test ./internal/tui/...
```

Tutti i comandi devono completarsi senza errori.





