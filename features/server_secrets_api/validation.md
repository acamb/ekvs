# validation.md — server_secrets_api

Descrive come verificare che l'implementazione della feature `server_secrets_api` sia corretta e completa.

---

## 1. Compilazione pulita

```bash
go build ./...
go vet ./...
```

Nessun errore di compilazione né warning da `go vet`. In particolare verificare che il refactor di `helpers.go` non abbia lasciato simboli duplicati o import orfani.

---

## 2. Unit Test

Eseguire:

```bash
go test ./internal/server/... -v -coverprofile=coverage.out
go tool cover -func=coverage.out | grep 'internal/server'
```

### Copertura attesa
- Package `internal/server`: **≥ 90%** degli statement.

### Scenari da coprire (table-driven)

#### `GET /projects/{name}`
| Scenario | Status atteso | Body atteso |
|----------|--------------|-------------|
| Progetto esiste, ha 2 segreti | `200` | `{"name":"p","keys":["a","b"]}` |
| Progetto esiste, nessun segreto | `200` | `{"name":"p","keys":[]}` |
| Progetto non esiste | `404` | `{"error":"..."}` |
| `userID` mancante nel contesto | `500` | `{"error":"..."}` |

#### `GET /projects/{name}/secrets`
| Scenario | Status atteso | Body atteso |
|----------|--------------|-------------|
| Progetto con segreti | `200` | `{"secrets":[{"key":"k1","value":"v1"},{"key":"k2","value":"v2"}]}` |
| Progetto vuoto | `200` | `{"secrets":[]}` |
| Progetto non esiste | `404` | `{"error":"..."}` |
| `userID` mancante | `500` | `{"error":"..."}` |

#### `PUT /projects/{name}/secrets/{key}`
| Scenario | Status atteso | Body atteso |
|----------|--------------|-------------|
| Creazione nuovo segreto | `200` | `{"key":"k"}` |
| Sovrascrittura segreto esistente | `200` | `{"key":"k"}` |
| Body JSON malformato | `400` | `{"error":"..."}` |
| Campo `value` assente | `400` | `{"error":"..."}` |
| Campo `value` stringa vuota | `400` | `{"error":"..."}` |
| Nome chiave invalido (es. `"bad name"`) | `400` | `{"error":"..."}` |
| Progetto non esiste | `404` | `{"error":"..."}` |
| `userID` mancante | `500` | `{"error":"..."}` |

#### `GET /projects/{name}/secrets/{key}`
| Scenario | Status atteso | Body atteso |
|----------|--------------|-------------|
| Segreto presente | `200` | `{"key":"k","value":"v"}` |
| Progetto non esiste | `404` | `{"error":"..."}` |
| Chiave non trovata | `404` | `{"error":"..."}` |
| `userID` mancante | `500` | `{"error":"..."}` |

#### `DELETE /projects/{name}/secrets/{key}`
| Scenario | Status atteso | Body atteso |
|----------|--------------|-------------|
| Eliminazione con successo | `204` | _(nessun corpo)_ |
| Progetto non esiste | `404` | `{"error":"..."}` |
| Chiave non trovata | `404` | `{"error":"..."}` |
| `userID` mancante | `500` | `{"error":"..."}` |

---

## 3. Test manuali curl

Avviare il server localmente con una configurazione di test (storage e chiavi in una directory temporanea).

### Setup
```bash
# Generare una chiave di test (se non già presente)
ssh-keygen -t ed25519 -f /tmp/test_key -N ""
cp /tmp/test_key.pub /tmp/ekvs_keys/

# Avviare il server
EKVS_STORAGE_PATH=/tmp/ekvs_storage EKVS_KEYS_DIR=/tmp/ekvs_keys ./ekvs-server
```

Per ogni richiesta autenticata, il client deve inviare gli header:
- `X-Fingerprint`: fingerprint SHA256 della chiave
- `X-Timestamp`: timestamp Unix corrente
- `X-Signature`: firma della stringa `<fingerprint>.<timestamp>` con la chiave privata

### Scenario completo
```bash
# 1. Creare un progetto
curl -s -X POST http://localhost:8080/projects/myproject \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 2. GET /projects/myproject (nessun segreto)
# Atteso: {"name":"myproject","keys":[]}
curl -s http://localhost:8080/projects/myproject \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 3. Set due segreti
curl -s -X PUT http://localhost:8080/projects/myproject/secrets/api_key \
  -H "Content-Type: application/json" \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..." \
  -d '{"value":"aGVsbG8="}'

curl -s -X PUT http://localhost:8080/projects/myproject/secrets/db_pass \
  -H "Content-Type: application/json" \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..." \
  -d '{"value":"d29ybGQ="}'

# 4. GET /projects/myproject (con segreti)
# Atteso: {"name":"myproject","keys":["api_key","db_pass"]}
curl -s http://localhost:8080/projects/myproject \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 5. GET /projects/myproject/secrets
# Atteso: {"secrets":[{"key":"db_pass","value":"d29ybGQ="}]}
# (api_key era già stato eliminato al passo 7)
curl -s http://localhost:8080/projects/myproject/secrets \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 6. GET singolo segreto
# Atteso: {"key":"api_key","value":"aGVsbG8="}
curl -s http://localhost:8080/projects/myproject/secrets/api_key \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 7. DELETE segreto
# Atteso: 204 No Content
curl -s -X DELETE http://localhost:8080/projects/myproject/secrets/api_key \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 8. GET dopo DELETE → 404
curl -s http://localhost:8080/projects/myproject/secrets/api_key \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# 9. Errori attesi
# Progetto inesistente → 404
curl -s http://localhost:8080/projects/nonexistent \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..."

# Nome chiave invalido → 400
curl -s -X PUT "http://localhost:8080/projects/myproject/secrets/bad name" \
  -H "Content-Type: application/json" \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..." \
  -d '{"value":"test"}'

# Body malformato → 400
curl -s -X PUT http://localhost:8080/projects/myproject/secrets/x \
  -H "Content-Type: application/json" \
  -H "X-Fingerprint: ..." -H "X-Timestamp: ..." -H "X-Signature: ..." \
  -d 'not-json'
```

---

## 4. No-regressione

Verificare che i test delle feature precedenti non abbiano regressioni dopo il refactor di `helpers.go`:

```bash
go test ./... -count=1
```

Tutti i test del progetto devono passare.

---

## 5. Criteri di accettazione

- [ ] `go build ./...` e `go vet ./...` senza errori.
- [ ] Tutti i test unitari passano (`go test ./...`).
- [ ] Copertura `internal/server` ≥ 90%.
- [ ] I 5 endpoint rispondono con i codici HTTP e i body JSON corretti negli scenari curl descritti.
- [ ] `keys` (in `GET /projects/{name}` e `GET /projects/{name}/secrets/{key}`) e `secrets` (in `GET /projects/{name}/secrets`) non sono mai `null` in nessuna risposta JSON (sempre array, eventualmente vuoti).
- [ ] `helpers.go` non contiene duplicati rispetto a `projects.go`.

