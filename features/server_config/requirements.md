# requirements.md — server_config

## User Decisions

| Decisione | Scelta |
|-----------|--------|
| Formato file | YAML (`ekvs.yaml`) — libreria `gopkg.in/yaml.v3` |
| Path di default | `ekvs.yaml` nella working directory del processo |
| Override via flag | `--config <path>` passato a `cmd/server/main.go`; parsato con `flag` della stdlib |
| Priorità | **env var > file YAML > default hardcoded** — coerente con il comportamento già esistente di `config.Load()` |
| File mancante | Se il file non esiste al path di default, il server si avvia con i valori di default (no errore). Se il file è specificato esplicitamente con `--config` e non esiste, il server termina con errore fatale |
| File malformato | YAML non parsabile → errore fatale con messaggio descrittivo |
| Creazione directory | All'avvio il server chiama `os.MkdirAll` su `StoragePath` e `KeysDir`; errore fatale se non riesce |
| Chiavi YAML | Corrispondono 1:1 ai campi di `Config`: `server_addr`, `storage_path`, `keys_dir`, `log_level` |
| Package interessati | `internal/config` (logica di caricamento), `cmd/server/main.go` (flag parsing + bootstrap) |
| Dipendenza esterna | `gopkg.in/yaml.v3` — da aggiungere con `go get` |

---

## Scope

### In scope
- Aggiungere `gopkg.in/yaml.v3` alle dipendenze del modulo.
- Nuova funzione `LoadFromFile(path string, required bool) (*Config, error)` in `internal/config`:
  - Legge il YAML, deserializza nei campi di `Config`.
  - Applica poi le env var come override.
  - Se `required=false` e il file non esiste, usa i soli default + env.
  - Se `required=true` e il file non esiste, restituisce errore.
- `Load()` esistente rimane invariato (compatibilità con i test esistenti).
- `cmd/server/main.go`: flag `--config`, chiamata a `LoadFromFile`, poi `ensureDirs`.
- Funzione interna `ensureDirs(cfg *Config, log logging.Logger) error` in `cmd/server/main.go` che chiama `os.MkdirAll` su `StoragePath` e `KeysDir`.
- Aggiornamento di `config_test.go` per coprire `LoadFromFile`.
- File di esempio `ekvs.yaml.example` nella root del repository.

### Out of scope
- Ricarica della configurazione a caldo (hot-reload).
- Validazione semantica dei valori (es. formato indirizzo): rimane responsabilità dei componenti che li usano.
- Sottocomandi CLI complessi — si usa solo `flag` stdlib.
- Cifratura del file di configurazione.

---

## Struttura YAML

```yaml
# ekvs.yaml — configurazione del server EKVS
server_addr:  "127.0.0.1:8080"
storage_path: "./data"
keys_dir:     "./data/.keys"
log_level:    "info"
```

---

## API aggiornata di `internal/config`

```go
package config

// LoadFromFile carica la configurazione da un file YAML a path.
// Se required è false e il file non esiste, usa default + env var.
// Se required è true e il file non esiste, restituisce errore.
// Le variabili d'ambiente hanno sempre precedenza sui valori del file.
func LoadFromFile(path string, required bool) (*Config, error)
```

---

## Priorità di risoluzione (esempio per `server_addr`)

```
EKVS_SERVER_ADDR (env)         → valore finale se impostata
  └─ server_addr in ekvs.yaml  → usato se env non impostata
       └─ "127.0.0.1:8080"     → default hardcoded
```

---

## Dipendenze

| Simbolo | Provenienza |
|---------|-------------|
| `gopkg.in/yaml.v3` | nuova dipendenza esterna |
| `logging.Logger` | `ekvs/internal/logging` |
| `config.Config` | `ekvs/internal/config` (aggiornato) |

