# validation.md — server_projects_api

## Criteri di accettazione

La feature è considerata completa quando **tutti** i seguenti criteri sono soddisfatti.

---

## 1. Unit test

```bash
make test
```

- Output: `ok  ekvs/internal/server` senza errori.
- Copertura statement ≥ 90% su `internal/server`.
- Nessuna race condition rilevata (i test girano con `-race`).

---

## 2. Verifica manuale con `curl`

Avviare il server:

```bash
EKVS_STORAGE_PATH=/tmp/ekvs-test \
EKVS_KEYS_DIR=/tmp/ekvs-test/.keys \
EKVS_SERVER_ADDR=:8080 \
go run ./cmd/server
```

Copiare la propria chiave pubblica nella directory `.keys/`:
```bash
mkdir -p /tmp/ekvs-test/.keys
cp ~/.ssh/id_ed25519.pub /tmp/ekvs-test/.keys/mykey.pub
```

> Le variabili `$TS`, `$FP`, `$SIG` vanno generate con lo script di firma sviluppato
> in `ssh_auth_primitives` / `server_auth` (firmare `METHOD /path timestamp`).

### Scenario A — Ciclo completo

```bash
# Creare un progetto
curl -s -X POST http://localhost:8080/projects/alpha \
  -H "X-Timestamp: $TS" -H "X-Fingerprint: $FP" -H "X-Signature: $SIG"
# Atteso: 201  {"name":"alpha"}

# Listare i progetti
curl -s http://localhost:8080/projects \
  -H "X-Timestamp: $TS" -H "X-Fingerprint: $FP" -H "X-Signature: $SIG"
# Atteso: 200  {"projects":["alpha"]}

# Eliminare il progetto
curl -s -X DELETE http://localhost:8080/projects/alpha \
  -H "X-Timestamp: $TS" -H "X-Fingerprint: $FP" -H "X-Signature: $SIG"
# Atteso: 204  (nessun corpo)

# Lista di nuovo
curl -s http://localhost:8080/projects \
  -H "X-Timestamp: $TS" -H "X-Fingerprint: $FP" -H "X-Signature: $SIG"
# Atteso: 200  {"projects":[]}
```

### Scenario B — Errori attesi

| Azione | Atteso |
|--------|--------|
| `POST /projects/alpha` due volte | Seconda risposta: `409 Conflict` |
| `POST /projects/!!bad!!` | `400 Bad Request` (nome invalido, caratteri non permessi) |
| `DELETE /projects/nonexistent` | `404 Not Found` |
| `DELETE /projects/!!bad!!` | `404 Not Found` (nome invalido non può esistere su disco) |
| Richiesta senza header di auth | `401 Unauthorized` (gestito da `AuthMiddleware`) |
| Timestamp scaduto (> 30 s) | `401 Unauthorized` |

---

## 3. Nessuna regressione

```bash
make test
```

Tutti i package già esistenti devono continuare a passare:
- `ekvs/internal/config` (incluso il nuovo campo `KeysDir`)
- `ekvs/internal/storage`
- `ekvs/internal/auth`
- `ekvs/internal/ssh`
- `ekvs/internal/encryption`
- `ekvs/internal/errors`
- `ekvs/internal/logging`

---

## 4. Compilazione del server

```bash
go build ./cmd/server
```

Deve completare senza errori o warning.

---

## 5. Isolation tra utenti

Verificato dagli unit test (`server_test.go`):
- UserID A crea `alpha` → UserID B lista → risposta `{"projects":[]}`.


