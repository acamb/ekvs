# plan.md — code_cleanup

Lista ordinata dei task atomici da eseguire. Ogni task è verificabile singolarmente con `go build ./...` e `go test ./...`.

---

### 1. Eliminare `internal/errors`

Rimuovere la directory `internal/errors/` con i file `errors.go` e `errors_test.go`. Verificare che nessun altro package li importi (`grep -r "ekvs/internal/errors"` deve dare output vuoto). Eseguire `go build ./...` per confermare.

---

### 2. Spostare la regex di `sanitizeID` a variabile di package

In `internal/storage/storage.go`, la funzione `sanitizeID` compila `regexp.MustCompile` ad ogni invocazione. Spostare la regex a variabile di package:
```go
var sanitizeRE = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
```
e aggiornare `sanitizeID` per usarla.

---

### 3. Correggere unhandled errors in `storage.writeProject`

In `internal/storage/storage.go`, nel corpo di `writeProject`:
- Riga 116 (defer cleanup): `os.Remove(tmpPath)` — aggiungere `_ =` con commento che spiega che il cleanup è best-effort e non deve mascherare l'errore originale.
- Riga 120 (`tmp.Close()` nel percorso errore di `Chmod`): aggiungere `_ =` con commento.
- Riga 124 (`tmp.Close()` nel percorso errore di `Write`): aggiungere `_ =` con commento.

---

### 4. Correggere `cipher.NewGCM` ignorato in `encryption/cipher.go`

Nelle funzioni `Encrypt` e `Decrypt` di `internal/encryption/cipher.go`, il pattern `gcm, _ := cipher.NewGCM(block)` ignora l'errore con `_`. Convertire in:
```go
gcm, err := cipher.NewGCM(block)
if err != nil {
    // unreachable: cipher.NewGCM only fails if block.BlockSize() != 16,
    // which cannot happen for an aes.NewCipher result.
    panic(fmt.Sprintf("cipher.NewGCM: %v", err))
}
```

---

### 5. Correggere `io.ReadFull` ignorato in `encryption/derive.go`

In `internal/encryption/derive.go` (riga 41), convertire `_, _ = io.ReadFull(r, key)` in:
```go
if _, err := io.ReadFull(r, key); err != nil {
    // unreachable: hkdf.New with sha256.New never returns an error from Read.
    panic(fmt.Sprintf("hkdf read: %v", err))
}
```

---

### 6. Correggere `buf.Write` non catturato in `encryption/derive.go`

In `internal/encryption/derive.go` (riga 68), aggiungere `_ =` prima di `buf.Write(p.Bytes())`:
```go
_ = buf.Write(p.Bytes()) // bytes.Buffer.Write never returns an error
```

---

### 7. Correggere `fmt.Sprintf("%s", err)` in `auth/middleware.go`

In `internal/auth/middleware.go`, sostituire:
- Riga 65: `fmt.Sprintf("%s", ErrInvalidSignature)` → `ErrInvalidSignature.Error()`
- Riga 71: `fmt.Sprintf("%s", ErrReplayDetected)` → `ErrReplayDetected.Error()`

---

### 8. Documentare gli errori `json.Encode` ignorati intenzionalmente

In `internal/server/helpers.go` (riga 12) e `internal/auth/middleware.go` (riga 101), aggiungere un commento esplicativo che giustifichi il `_ =` prima dell'`Encode`:
```go
// Encoding errors are intentionally ignored: the response header has already
// been written via WriteHeader, so there is nothing useful to do on failure.
_ = json.NewEncoder(w).Encode(...)
```

---

### 9. Aggiungere helper per errori ignorati nei test `auth`

In `internal/auth/auth_test.go`, introdurre helper locali:
```go
func mustNewKeyStore(t *testing.T, dir string) *KeyStore { ... }
func mustSign(t *testing.T, signer crypto.Signer, msg []byte) []byte { ... }
```
e usarli al posto dei pattern `x, _ := f()` e `signer, _, _ := ParsePrivateKey(...)`. Questo elimina i warning e rende i test robusti rispetto a fallimenti inattesi.

---

### 10. Correggere return value completamente ignorati in `TestConcurrency`

In `internal/auth/auth_test.go`, nel goroutine di `TestConcurrency`, le chiamate `ks.Lookup(fpRsa)` e `ks.List()` ignorano i return value senza `_`. Convertire in:
```go
_, _ = ks.Lookup(fpRsa) // best-effort mix; errors checked in the dedicated goroutines
_, _ = ks.List()
```

---

### 11. Aggiungere test per `storage.writeProject` — percorsi di errore

In `internal/storage/storage_test.go`, aggiungere casi che coprono:
- Fallimento di `os.CreateTemp`: rendere la directory utente non scrivibile prima di chiamare `SetSecret`.
- Fallimento di `os.Rename`: creare il progetto, poi rendere la directory non scrivibile dopo la creazione del file temporaneo ma prima del rename (usando `os.Chmod 0o500` sulla directory padre).

---

### 12. Aggiungere test per `ssh.ParsePrivateKey` — percorsi di errore

In `internal/ssh/ssh_test.go`, aggiungere casi per:
- PEM con tipo non riconosciuto (es. blocco `CERTIFICATE` o `DSA PRIVATE KEY`).
- Dati PEM validi ma chiave di tipo non supportato che supera il parsing di `gossh.ParseRawPrivateKey` ma fallisce lo switch di tipo.

---

### 13. Aggiungere test per `server.handleListSecrets` — errore interno di `GetSecret`

In `internal/server/secrets_test.go`, aggiungere un test che copra il percorso 500 dell'handler `GET /projects/{name}/secrets` quando `GetSecret` fallisce dopo che `ListSecrets` ha avuto successo. Approccio: creare progetto con segreti, poi corrompere il file di progetto su disco (sovrascrivere con JSON invalido o rendere il file illeggibile) prima di chiamare l'handler.

---

### 14. Audit uniformità gestione errori — server

Leggere tutti gli handler in `internal/server/projects.go` e `internal/server/secrets.go` e verificare che:
- Ogni `500` sia preceduto da `log.Error(...)`.
- Nessun `4xx` venga loggato con `.Error(...)`.
- I messaggi di errore JSON siano consistenti.
Apportare le correzioni necessarie.

---

### 15. Audit uniformità logging

Verificare che le chiavi dei log siano uniformi in tutti gli handler (`"user"`, `"project"`, `"key"`, `"error"`, `"count"`). Apportare le correzioni necessarie.

---

### 16. Esecuzione `go vet ./...`

Eseguire `go vet ./...` e correggere tutti i warning.

---

### 17. Verifica finale

```bash
go build ./...
go vet ./...
go test ./... -count=1 -coverprofile=coverage.out
go tool cover -func=coverage.out
```
Verificare che tutti i package `internal/*` abbiano copertura ≥ 90% e che tutti i test passino.
