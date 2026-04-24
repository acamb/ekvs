# plan.md — server_config

## Ordered Task List

1. **Aggiungere `gopkg.in/yaml.v3`**
   ```bash
   go get gopkg.in/yaml.v3
   go mod tidy
   ```

2. **Aggiungere struct YAML e `LoadFromFile` a `internal/config/config.go`**

   Aggiungere una struct interna per il mapping YAML:
   ```go
   type yamlConfig struct {
       ServerAddr  string `yaml:"server_addr"`
       StoragePath string `yaml:"storage_path"`
       KeysDir     string `yaml:"keys_dir"`
       LogLevel    string `yaml:"log_level"`
   }
   ```

   Implementare `LoadFromFile(path string, required bool) (*Config, error)`:
   1. `os.ReadFile(path)`:
      - Se `os.IsNotExist` e `!required` → usa solo default + env (chiama `Load()`).
      - Se `os.IsNotExist` e `required` → restituisce errore.
      - Altro errore → restituisce errore.
   2. `yaml.Unmarshal(data, &yc)` → errore se malformato.
   3. Costruire `Config` con precedenza: env var > valore YAML > default hardcoded.
      Usare un helper interno:
      ```go
      func coalesce(envKey, yamlVal, fallback string) string
      ```
      che restituisce: `envOr(envKey, yamlVal)` se `yamlVal != ""`, altrimenti `envOr(envKey, fallback)`.

3. **Aggiornare `config_test.go`**

   Aggiungere test table-driven per `LoadFromFile`:
   - File valido con tutti i campi → valori dal file.
   - File valido + env var impostata → env var vince.
   - File non esiste + `required=false` → default/env, no errore.
   - File non esiste + `required=true` → errore.
   - File YAML malformato → errore.
   - File con campi parziali → campi mancanti usano default.

4. **Aggiornare `cmd/server/main.go`**

   Aggiungere flag parsing all'inizio di `main()`:
   ```go
   configPath := flag.String("config", "ekvs.yaml", "path to YAML config file")
   flag.Parse()
   ```

   Sostituire `config.Load()` con:
   ```go
   cfg, err := config.LoadFromFile(*configPath, *configPath != "ekvs.yaml")
   ```
   > Se il path è quello di default e il file non esiste, non è un errore.
   > Se l'utente specifica esplicitamente `--config`, il file deve esistere.

   Aggiungere `ensureDirs` dopo il caricamento della configurazione:
   ```go
   func ensureDirs(cfg *config.Config, log logging.Logger) error {
       for _, dir := range []string{cfg.StoragePath, cfg.KeysDir} {
           if err := os.MkdirAll(dir, 0700); err != nil {
               log.Error("failed to create directory", "dir", dir, "error", err)
               return err
           }
       }
       return nil
   }
   ```
   Chiamarla subito dopo aver creato il logger, prima di `storage.New` e `auth.NewKeyStore`.

5. **Creare `ekvs.yaml.example`** nella root del repository:
   ```yaml
   # ekvs.yaml — example configuration for the EKVS server
   # Copy to ekvs.yaml and edit as needed.
   # Environment variables (EKVS_SERVER_ADDR, etc.) override these values.
   server_addr:  "127.0.0.1:8080"
   storage_path: "./data"
   keys_dir:     "./data/.keys"
   log_level:    "info"
   ```

6. **Eseguire `go mod tidy` e validare**
   ```bash
   go mod tidy
   make test   # tutti i package devono passare
   go build ./cmd/server
   ```
   Confermare copertura ≥ 90% su `internal/config`.

