# validation.md — server_config

## Criteri di accettazione

La feature è considerata completa quando **tutti** i seguenti criteri sono soddisfatti.

---

## 1. Unit test

```bash
make test
```

- Output: `ok  ekvs/internal/config` senza errori.
- Copertura statement ≥ 90% su `internal/config`.
- Nessuna regressione sugli altri package.

---

## 2. Verifica manuale — caricamento da file YAML

Creare un file di configurazione:
```bash
cat > /tmp/test-ekvs.yaml <<EOF
server_addr:  "127.0.0.1:9090"
storage_path: "/tmp/ekvs-data"
keys_dir:     "/tmp/ekvs-data/.keys"
log_level:    "debug"
EOF
```

Avviare il server con `--config`:
```bash
go run ./cmd/server --config /tmp/test-ekvs.yaml
```

Verificare:
- Il server si avvia sull'indirizzo `127.0.0.1:9090` (non quello di default 8080).
- Le directory `/tmp/ekvs-data` e `/tmp/ekvs-data/.keys` vengono create automaticamente.
- Il log mostra messaggi a livello `debug`.

---

## 3. Verifica — file di default assente

```bash
cd /tmp && go run /path/to/ekvs/cmd/server
```
Il server deve avviarsi con i valori di default senza errori (il file `ekvs.yaml` non esiste nella CWD).

---

## 4. Verifica — `--config` esplicito su file mancante

```bash
go run ./cmd/server --config /tmp/nonexistent.yaml
```
Il server deve terminare con un errore fatale descrittivo e codice di uscita non zero.

---

## 5. Verifica — priorità env var > file

```bash
EKVS_SERVER_ADDR=127.0.0.1:7777 go run ./cmd/server --config /tmp/test-ekvs.yaml
```
Il server deve avviarsi su `127.0.0.1:7777` (env var) anche se il file dice `9090`.

---

## 6. Verifica — creazione directory mancanti

```bash
rm -rf /tmp/ekvs-autodir
cat > /tmp/autodir.yaml <<EOF
storage_path: "/tmp/ekvs-autodir/data"
keys_dir:     "/tmp/ekvs-autodir/data/.keys"
EOF
go run ./cmd/server --config /tmp/autodir.yaml &
sleep 1
ls /tmp/ekvs-autodir/data/.keys   # deve esistere
kill %1
```

---

## 7. Compilazione

```bash
go build ./cmd/server
```
Deve completare senza errori.

---

## 8. Nessuna regressione

```bash
make test
```

Tutti i package devono passare:
- `ekvs/internal/config`
- `ekvs/internal/server`
- `ekvs/internal/auth`
- `ekvs/internal/storage`
- `ekvs/internal/ssh`
- `ekvs/internal/encryption`
- `ekvs/internal/errors`
- `ekvs/internal/logging`

