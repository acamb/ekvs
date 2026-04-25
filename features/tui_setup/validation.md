# validation.md — tui_setup

## Criteri di accettazione

La feature è considerata completa quando **tutti** i seguenti criteri sono soddisfatti.

---

## 1. Unit test

```bash
make test
```

- Output: `ok  ekvs/internal/tui/config` e `ok  ekvs/internal/tui/theme` senza errori.
- Copertura statement ≥ 90% su `internal/tui/config` e `internal/tui/theme`.
- Nessuna regressione sugli altri package (`internal/auth`, `internal/config`, `internal/encryption`, `internal/ssh`, `internal/storage`, `internal/server`).

Oppure in modo mirato:

```bash
go test ./internal/tui/... -v -cover
```

---

## 2. Compilazione

```bash
go build ./cmd/tui/...
```

Deve completarsi senza errori né warning.

---

## 3. Avvio con file di configurazione valido — profilo singolo

Creare un file di configurazione con un solo profilo:

```bash
cat > /tmp/test-ekvs-tui.yaml <<EOF
profiles:
  - name:          "test"
    server_url:    "http://127.0.0.1:9090"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "hacker"
EOF
```

Avviare il TUI:

```bash
go run ./cmd/tui --config /tmp/test-ekvs-tui.yaml
```

Verificare:
- La schermata di selezione profilo **non** viene mostrata (un solo profilo → avvio diretto).
- Il TUI si avvia e mostra il menu principale con sfondo nero e testo verde (tema `hacker`).
- Le quattro voci `Projects`, `Secrets`, `Settings`, `Quit` sono visibili.
- La prima voce è evidenziata con il cursore visivo.

---

## 4. Navigazione del menu

Con il TUI avviato (qualsiasi tema):

| Azione | Comportamento atteso |
|--------|----------------------|
| Tasto `↓` o `j` | Il cursore si sposta alla voce successiva |
| Tasto `↑` o `k` | Il cursore si sposta alla voce precedente |
| `↑` dalla prima voce | Wrap-around: il cursore va all'ultima voce |
| `↓` dall'ultima voce | Wrap-around: il cursore va alla prima voce |
| `Enter` su `Quit` | L'applicazione termina senza errori |
| `q` da qualsiasi voce | L'applicazione termina senza errori |
| `Ctrl+C` | L'applicazione termina senza errori |
| `Enter` su `Projects`, `Secrets`, `Settings` | Nessun crash (comportamento placeholder) |

---

## 5. Tema adattativo

Creare un file con `theme: "adaptive"` e avviarlo:

```bash
cat > /tmp/adaptive.yaml <<EOF
profiles:
  - name: "locale"
    server_url: "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme: "adaptive"
EOF
go run ./cmd/tui --config /tmp/adaptive.yaml
```

Il TUI deve adattarsi al tema del terminale. Verifica visiva.

---

## 6. Selezione profilo — file con più profili

Creare un file con due profili:

```bash
cat > /tmp/multi.yaml <<EOF
profiles:
  - name:          "locale"
    server_url:    "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "adaptive"
  - name:          "produzione"
    server_url:    "https://ekvs.example.com"
    identity_file: "~/.ssh/id_rsa"
    theme:         "hacker"
EOF

go run ./cmd/tui --config /tmp/multi.yaml
```

Verificare:
- Viene mostrata la schermata di selezione profilo **prima** del menu principale.
- Ogni riga mostra il nome e l'URL del profilo.
- Navigazione con `↑↓` e selezione con `Enter`.
- Scegliendo `locale` → menu principale con tema `adaptive`.
- Scegliendo `produzione` → menu principale con tema `hacker`.
- Premendo `q`/`Ctrl+C` nella schermata di selezione → uscita dall'applicazione.

---

## 7. Nomi duplicati nel file di configurazione

```bash
cat > /tmp/dup.yaml <<EOF
profiles:
  - name: "locale"
    server_url: "http://127.0.0.1:8080"
  - name: "locale"
    server_url: "http://127.0.0.1:9090"
EOF

go run ./cmd/tui --config /tmp/dup.yaml
```

Il programma deve terminare con errore fatale che indica il nome duplicato. Exit code non zero.

---

## 8. Lista profili vuota

```bash
cat > /tmp/empty.yaml <<EOF
profiles: []
EOF

go run ./cmd/tui --config /tmp/empty.yaml
```

Il wizard di primo avvio deve essere mostrato (comportamento identico al file assente).

---

## 9. Tema non riconosciuto

```bash
cat > /tmp/bad-theme.yaml <<EOF
profiles:
  - name: "test"
    server_url: "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme: "invalid_theme"
EOF

go run ./cmd/tui --config /tmp/bad-theme.yaml
```

Il programma deve terminare con errore fatale e messaggio che indica il nome del tema non riconosciuto. Exit code non zero.

---

## 10. Flag `--config` su file mancante

```bash
go run ./cmd/tui --config /tmp/nonexistent-config.yaml
```

Il programma deve terminare con errore fatale e messaggio descrittivo. Exit code non zero.

---

## 11. Wizard di primo avvio — file di default assente

Spostarsi in una directory temporanea dove `ekvs-tui.yaml` non esiste:

```bash
mkdir -p /tmp/ekvs-test-dir && cd /tmp/ekvs-test-dir
go run /home/andrea/src/ekvs/cmd/tui
```

Verificare:
- Il TUI non si avvia direttamente, ma compare il wizard interattivo.
- Il wizard mostra un campo per `name` (vuoto), poi campi pre-compilati per `server_url` (default: `http://127.0.0.1:8080`) e `identity_file`.
- Dopo aver completato tutti i campi, viene chiesto se salvare la configurazione.

---

## 12. Wizard — salvataggio configurazione

Proseguendo dal punto 11:

- Rispondere `s` alla domanda di salvataggio.
- Viene chiesto il nome del file (default: `ekvs-tui.yaml`).
- Premere `Enter` per accettare il default.
- Verificare che il file `ekvs-tui.yaml` sia stato creato nella directory corrente con i valori inseriti nella struttura a profili.
- Verificare che il TUI si avvii normalmente dopo il salvataggio.

```bash
cat /tmp/ekvs-test-dir/ekvs-tui.yaml
```

Il contenuto deve rispecchiare i valori inseriti nel wizard, nel formato:
```yaml
profiles:
  - name: "..."
    server_url: "..."
    identity_file: "..."
    theme: "adaptive"
```

---

## 13. Wizard — skip salvataggio

Ripetere il punto 11 rispondendo `n` alla domanda di salvataggio:

- Il TUI si avvia normalmente senza creare alcun file.
- Al successivo avvio nella stessa directory, il wizard viene ripresentato (nessun file salvato).

---

## 14. File YAML malformato

```bash
echo "questo: non: è: yaml: valido: [" > /tmp/malformed.yaml
go run ./cmd/tui --config /tmp/malformed.yaml
```

Il programma deve terminare con errore fatale e messaggio che indica il problema di parsing YAML. Exit code non zero.

---

## 15. File di esempio

Verificare che `ekvs-tui.yaml.example` esista nella root del repository e contenga almeno due profili di esempio con tutti i campi (`name`, `server_url`, `identity_file`, `theme`).

```bash
cat ekvs-tui.yaml.example
```






