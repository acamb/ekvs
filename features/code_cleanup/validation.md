# validation.md — code_cleanup

Descrive come verificare che il cleanup sia completo e corretto.

---

## 1. Compilazione pulita

```bash
go build ./...
go vet ./...
```

Nessun errore né warning. Se `staticcheck` è disponibile:
```bash
staticcheck ./...
```

---

## 2. `internal/errors` eliminato

```bash
ls internal/errors/
# deve restituire: No such file or directory

grep -r "ekvs/internal/errors" --include="*.go" .
# deve restituire: nessun output
```

---

## 3. Tutti i test passano

```bash
go test ./... -count=1
```

Output atteso: tutti i package `ok`, nessun `FAIL`.

---

## 4. Copertura ≥ 90% per tutti i package `internal/*`

```bash
go test ./internal/... -coverprofile=coverage.out -count=1
go tool cover -func=coverage.out
```

Soglie minime:

| Package | Target |
|---------|--------|
| `internal/auth` | ≥ 90% |
| `internal/config` | ≥ 90% |
| `internal/encryption` | ≥ 90% |
| `internal/errors` | eliminato |
| `internal/logging` | ≥ 90% |
| `internal/server` | ≥ 90% |
| `internal/ssh` | ≥ 90% |
| `internal/storage` | ≥ 90% |

---

## 5. Verifica warning corretti — checklist per file

### `internal/storage/storage.go`
- [ ] `sanitizeRE` è una variabile di package — `sanitizeID` non compila più una regex ad ogni chiamata.
- [ ] `os.Remove(tmpPath)` nel defer ha `_ =` con commento.
- [ ] `tmp.Close()` a riga 120 ha `_ =` con commento.
- [ ] `tmp.Close()` a riga 124 ha `_ =` con commento.

### `internal/encryption/cipher.go`
- [ ] `cipher.NewGCM(block)` in `Encrypt` — errore gestito con `panic` documentato.
- [ ] `cipher.NewGCM(block)` in `Decrypt` — errore gestito con `panic` documentato.

### `internal/encryption/derive.go`
- [ ] `io.ReadFull(r, key)` — convertito in `if _, err := ...; err != nil { panic(...) }` con commento.
- [ ] `buf.Write(p.Bytes())` — `_ =` aggiunto con commento.

### `internal/auth/middleware.go`
- [ ] `fmt.Sprintf("%s", ErrInvalidSignature)` sostituito con `ErrInvalidSignature.Error()`.
- [ ] `fmt.Sprintf("%s", ErrReplayDetected)` sostituito con `ErrReplayDetected.Error()`.
- [ ] `_ = json.NewEncoder(w).Encode(...)` ha commento esplicativo sull'intenzionalità.

### `internal/server/helpers.go`
- [ ] `_ = json.NewEncoder(w).Encode(v)` ha commento esplicativo sull'intenzionalità.

### `internal/auth/auth_test.go`
- [ ] Nessun pattern `x, _ := f()` per errori di setup — sostituiti da helper `mustNewKeyStore`, `mustSign`, ecc.
- [ ] In `TestConcurrency`, le chiamate `ks.Lookup(fpRsa)` e `ks.List()` usano `_, _ =`.

---

## 6. Audit errori e logging — checklist manuale

Leggere `internal/server/projects.go`, `secrets.go`, `helpers.go` e verificare:

- [ ] Ogni risposta `500` è preceduta da `log.Error(...)`.
- [ ] Nessuna risposta `4xx` è loggata con `log.Error(...)`.
- [ ] Tutti i messaggi JSON di errore usano la chiave `"error"`.
- [ ] Nessun handler usa `fmt.Println` o `fmt.Printf`.
- [ ] Le chiavi di log sono uniformi: `"user"`, `"project"`, `"key"`, `"error"`, `"count"`.

---

## 7. Criteri di accettazione

- [ ] `go build ./...` senza errori.
- [ ] `go vet ./...` senza warning.
- [ ] `go test ./... -count=1` — tutti i test passano.
- [ ] Tutti i package `internal/*` hanno copertura ≥ 90%.
- [ ] La directory `internal/errors/` non esiste più.
- [ ] Nessun file `.go` importa `ekvs/internal/errors`.
- [ ] `sanitizeID` usa una variabile di package per la regex, non `regexp.MustCompile` inline.
- [ ] Nessun `fmt.Sprintf("%s", err)` residuo — sostituito con `.Error()`.
- [ ] Nessun `tmp.Close()` senza `_ =` e commento nei percorsi di errore.
- [ ] `os.Remove` nel defer di `writeProject` ha `_ =` con commento.
- [ ] `cipher.NewGCM` gestito con `panic` documentato in `Encrypt` e `Decrypt`.
- [ ] `io.ReadFull` in `derive.go` gestito con `panic` documentato.
- [ ] `buf.Write` in `derive.go` ha `_ =` con commento.
- [ ] `json.Encode` ignorati in `helpers.go` e `middleware.go` hanno commento esplicativo.
- [ ] Nessun pattern `x, _ := f()` per errori di setup nei test `auth` — sostituiti da helper.
- [ ] `TestConcurrency` usa `_, _ =` per le chiamate senza cattura del return.
- [ ] Audit logging e error handling completato senza discrepanze residue.




