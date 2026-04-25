# requirements.md — tui_setup

## Decisioni utente

| Decisione | Scelta |
|-----------|--------|
| Navigazione | Menu principale a lista selezionabile (frecce su/giù + Invio) |
| File di configurazione | `ekvs-tui.yaml` nella directory corrente (default); override con `--config <path>` |
| File assente (path di default) | Wizard interattivo: chiede server URL e identity file, offre salvataggio con scelta nome |
| File assente (`--config` esplicito) | Errore fatale con messaggio descrittivo, exit code non zero |
| File malformato | Errore fatale con messaggio descrittivo |
| Tema di default | Adattativo (`adaptive`, usa `lipgloss.AdaptiveColor`) |
| Temi disponibili | `adaptive` (default), `hacker` (verde su nero) |
| Selezione tema | Campo `theme` nel file YAML; valore non riconosciuto → errore fatale |
| Librerie UI | `charm.land/bubbletea/v2 v2.0.6`, `github.com/charmbracelet/bubbles v1.0.0`, `github.com/charmbracelet/lipgloss v1.1.0` |
| Profili multipli | Il file YAML può contenere una lista di profili, ognuno con `name`, `server_url`, `identity_file`, `theme` |
| Selezione profilo | Se il file contiene un solo profilo, viene usato direttamente; se ne contiene più d'uno, all'avvio viene mostrata una schermata di selezione profilo |
| Flag `--config` | Incluso in questa fase; parsato con `flag` stdlib |
| Package interni | `internal/tui/config`, `internal/tui/theme`, `internal/tui/app` |

---

## Scope

### In scope
- Aggiunta dipendenze `charm.land/bubbletea/v2`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`.
- Package `internal/tui/config`: struct `Profile` e `ConfigFile`, funzioni `LoadFromFile` e `Save`, logica di merge con i default.
- Package `internal/tui/theme`: interfaccia `Theme`, implementazioni `AdaptiveTheme` e `HackerTheme`, factory `NewTheme(name string) (Theme, error)`.
- Package `internal/tui/app`: modello bubbletea principale con menu a lista (voci placeholder), navigazione tastiera.
- Schermata di selezione profilo (bubbletea, usa `bubbles/list`) mostrata all'avvio se il file contiene più di un profilo.
- Wizard di primo avvio: raccolta interattiva di `name`, `server_url` e `identity_file` tramite `bubbles/textinput`, offerta di salvataggio su file.
- `cmd/tui/main.go`: flag `--config`, bootstrap completo (caricamento file → wizard se file assente → selezione profilo se più d'uno → avvio app bubbletea).
- Unit test per `internal/tui/config` e `internal/tui/theme`.
- File di esempio `ekvs-tui.yaml.example` nella root del repository.

### Out of scope
- Chiamate HTTP al server (rinviate a `tui_auth` e successive).
- Autenticazione SSH (rinviata a `tui_auth`).
- Schermata Projects, Secrets, Settings (schermate placeholder sufficienti).
- Hot-reload della configurazione.
- Validazione semantica di `server_url` e `identity_file` (es. raggiungibilità server, esistenza file).
- CRUD profili dall'interno dell'applicazione (aggiungere/rimuovere profili a runtime).

---

## Struttura YAML — `ekvs-tui.yaml`

Il file contiene una lista di profili. Ogni profilo ha un `name` univoco e i propri parametri di connessione. Il campo `theme` è per-profilo: permette temi diversi per server diversi.

```yaml
# ekvs-tui.yaml — configurazione del client TUI EKVS
profiles:
  - name:          "locale"
    server_url:    "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "adaptive"

  - name:          "produzione"
    server_url:    "https://ekvs.example.com"
    identity_file: "~/.ssh/id_rsa"
    theme:         "hacker"
```

**Regole:**
- `name` deve essere non vuoto e univoco all'interno del file; se due profili hanno lo stesso nome, il caricamento restituisce errore.
- Se `theme` è omesso in un profilo, viene usato il default `"adaptive"`.
- Se `server_url` o `identity_file` sono omessi, vengono usati i rispettivi default.
- Un file con lista `profiles` vuota è equivalente a un file assente (viene avviato il wizard).

---

## Package `internal/tui/config`

```go
package config

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

// DefaultProfile restituisce un Profile con i valori di default.
// Usato come base per il merge e come valore iniziale del wizard.
func DefaultProfile() Profile

// LoadFromFile carica il ConfigFile dal file YAML a path.
//   - Se required è false e il file non esiste, restituisce (nil, nil)
//     per segnalare che il wizard deve essere avviato.
//   - Se required è true e il file non esiste, restituisce errore.
//   - Se il file è malformato YAML, restituisce errore.
//   - Se due profili hanno lo stesso name, restituisce errore.
//   - Se la lista profiles è vuota, restituisce (nil, nil) come nel caso file assente.
func LoadFromFile(path string, required bool) (*ConfigFile, error)

// Save serializza cf in YAML e scrive il file a path.
func Save(path string, cf *ConfigFile) error
```

**Default per ogni `Profile`:**

| Campo | Default |
|-------|---------|
| `ServerURL` | `"http://127.0.0.1:8080"` |
| `IdentityFile` | `"~/.ssh/id_ed25519"` |
| `Theme` | `"adaptive"` |

I campi omessi in un profilo del file YAML vengono riempiti con i valori di `DefaultProfile()` dopo l'unmarshal.

---

## Package `internal/tui/theme`

```go
package theme

import "github.com/charmbracelet/lipgloss"

// Theme raggruppa tutti gli stili lipgloss usati dall'applicazione.
type Theme interface {
    // Colori semantici
    PrimaryColor() lipgloss.Color
    SecondaryColor() lipgloss.Color
    BackgroundColor() lipgloss.Color
    ErrorColor() lipgloss.Color

    // Stili predefiniti
    TitleStyle() lipgloss.Style
    MenuItemStyle() lipgloss.Style
    SelectedMenuItemStyle() lipgloss.Style
    StatusBarStyle() lipgloss.Style
    ErrorStyle() lipgloss.Style
}

// NewTheme restituisce il Theme corrispondente al nome fornito.
// Nomi validi: "adaptive", "hacker".
// Restituisce errore per nomi non riconosciuti.
func NewTheme(name string) (Theme, error)
```

**Temi:**
- `AdaptiveTheme`: usa `lipgloss.AdaptiveColor{Light: "<light>", Dark: "<dark>"}` per adattarsi al terminale.
- `HackerTheme`: verde `#00FF41` su sfondo nero `#0D0208`, ispirato al Matrix.

---

## Package `internal/tui/app`

```go
package app

import (
    tea "charm.land/bubbletea/v2"
    "ekvs/internal/tui/theme"
)

// MenuItem rappresenta una voce del menu principale.
type MenuItem struct {
    Label string
    ID    string
}

// Model è il modello bubbletea principale dell'applicazione.
type Model struct {
    items    []MenuItem
    cursor   int
    theme    theme.Theme
    quitting bool
}

// New crea un nuovo Model con il tema fornito.
func New(t theme.Theme) Model

// Init, Update, View implementano l'interfaccia tea.Model.
func (m Model) Init() (Model, tea.Cmd)
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() string
```

**Voci menu placeholder (in ordine):**

| ID | Label |
|----|-------|
| `projects` | `Projects` |
| `secrets` | `Secrets` |
| `settings` | `Settings` |
| `quit` | `Quit` |

**Tasti:**
- `↑` / `k`: voce precedente
- `↓` / `j`: voce successiva
- `Enter`: seleziona voce (placeholder: stampa il nome, in fasi successive navigherà verso la schermata corrispondente)
- `q` / `Ctrl+C`: esce dall'applicazione

---

## `cmd/tui/main.go`

```go
// Flag supportati:
//   --config <path>   percorso al file di configurazione YAML (default: "ekvs-tui.yaml")
//
// Logica di bootstrap:
//  1. Parsare i flag.
//  2. Determinare se il path di config è quello di default o esplicito.
//  3. Chiamare config.LoadFromFile(path, explicit).
//     - (nil, nil)          → file assente o profili vuoti: avviare il wizard di primo avvio.
//     - (nil, err)          → file mancante (explicit=true) o malformato: log.Fatal.
//     - (cf, nil) con 1 profilo  → usa direttamente quel profilo.
//     - (cf, nil) con N profili  → mostra la schermata di selezione profilo.
//  4. Chiamare theme.NewTheme(profile.Theme) → errore fatale se nome non riconosciuto.
//  5. Creare app.New(t) e avviare tea.NewProgram(model).Run().
```

**Wizard di primo avvio** (package `internal/tui/wizard`):
- Raccoglie `name`, `server_url` e `identity_file` tramite `bubbles/textinput` (pre-compilati con i default).
- Chiede: `Salvare la configurazione? [s/N]`.
- Se sì, chiede il nome del file (default: `ekvs-tui.yaml`) e chiama `config.Save(path, &ConfigFile{Profiles: []Profile{p}})`.
- Restituisce il `Profile` raccolto.

**Schermata di selezione profilo** (package `internal/tui/profileselect` o come modello in `internal/tui/app`):
- Lista dei profili con `name` e `server_url` visibili.
- Navigazione `↑↓/jk`, `Enter` per selezionare, `q`/`Ctrl+C` per uscire.
- Restituisce il `Profile` scelto.

---

## Dipendenze da aggiungere

```bash
go get charm.land/bubbletea/v2@v2.0.6
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/charmbracelet/lipgloss@v1.1.0
go mod tidy
```






