# requirements.md — code_cleanup

## Obiettivo

Rifattorizzare e riordinare il codebase del server al termine della Phase 1, rimuovendo scaffolding temporaneo, uniformando la gestione degli errori e del logging, eliminando dead code, e portando la copertura dei test delle funzioni non ancora al 100% ai livelli stabiliti dalla policy.

Questa milestone **non introduce nuove funzionalità**. Non deve cambiare nessun contratto HTTP pubblico né nessuna firma di funzione pubblica.

---

## User Decisions

| Decisione | Scelta |
|-----------|--------|
| `internal/errors` | Il package è dead code: nessun package interno lo importa. Va **eliminato** (file `errors.go`, `errors_test.go` e la directory). |
| `auth.NewContextWithUserID` | La funzione è pubblica ma usata solo nei test dei package `server` e `auth`. Va mantenuta (è parte dell'API pubblica di `internal/auth` usata dai test). |
| Copertura minima per package | Tutti i package `internal/*` devono raggiungere **≥ 90%** di copertura statement. |
| `storage.writeProject` (63.6%) | Aggiungere test per i percorsi di errore (`chmod`, `write`, `close`, `rename`) usando un approccio compatibile con i permessi del filesystem. |
| `internal/server/secrets.go` `registerSecretsRoutes` (85.9%) | Aggiungere test per il percorso di errore interno di `handleListSecrets` (fallimento di `GetSecret` dopo `ListSecrets`) usando uno store corrotto. |
| `internal/ssh` `ParsePrivateKey` (80%), `Sign` (85.7%) | Aggiungere casi di test per i percorsi di errore: chiave non supportata, reader fallace, ecc. |
| Licenza/header di file | Non richiesto. |
| Nessuna modifica alle API HTTP | Confermato: zero cambiamenti ai contratti REST. |

---

## Warning concreti trovati nel codebase

### Codice di produzione

| File | Riga | Categoria | Descrizione |
|------|------|-----------|-------------|
| `internal/storage/storage.go` | 116 | unhandled error | `os.Remove(tmpPath)` nel defer — return value ignorato implicitamente |
| `internal/storage/storage.go` | 120 | unhandled error | `tmp.Close()` nel percorso di errore `Chmod` — return value ignorato |
| `internal/storage/storage.go` | 124 | unhandled error | `tmp.Close()` nel percorso di errore `Write` — return value ignorato |
| `internal/storage/storage.go` | `sanitizeID` | performance / bad practice | La funzione compila un `regexp.MustCompile` ad ogni chiamata — va spostato a variabile di package |
| `internal/encryption/cipher.go` | 25 | unhandled error | `gcm, _ := cipher.NewGCM(block)` in `Encrypt` — errore ignorato con `_` |
| `internal/encryption/cipher.go` | 57 | unhandled error | `gcm, _ := cipher.NewGCM(block)` in `Decrypt` — errore ignorato con `_` |
| `internal/encryption/derive.go` | 41 | unhandled error | `_, _ = io.ReadFull(r, key)` — entrambi i return value ignorati |
| `internal/encryption/derive.go` | 68 | unhandled error | `buf.Write(p.Bytes())` — return value non catturato (anche se `bytes.Buffer` non fallisce mai) |
| `internal/auth/middleware.go` | 65 | codice ridondante | `fmt.Sprintf("%s", ErrInvalidSignature)` — va sostituito con `ErrInvalidSignature.Error()` |
| `internal/auth/middleware.go` | 71 | codice ridondante | `fmt.Sprintf("%s", ErrReplayDetected)` — va sostituito con `ErrReplayDetected.Error()` |
| `internal/auth/middleware.go` | 101 | unhandled error | `_ = json.NewEncoder(w).Encode(...)` — errore ignorato, manca commento esplicativo |
| `internal/server/helpers.go` | 12 | unhandled error | `_ = json.NewEncoder(w).Encode(v)` — errore ignorato, manca commento esplicativo |

### Codice di test

| File | Riga approx. | Categoria | Descrizione |
|------|-------------|-----------|-------------|
| `internal/auth/auth_test.go` | 297 | unhandled error | `ks, _ := NewKeyStore(dir)` — errore ignorato con `_` |
| `internal/auth/auth_test.go` | 299 | unhandled error | `signerData, _ := os.ReadFile(...)` — errore ignorato con `_` |
| `internal/auth/auth_test.go` | 300 | unhandled error | `signer, _, _ := internalssh.ParsePrivateKey(signerData)` — errori ignorati con `_` |
| `internal/auth/auth_test.go` | ~453–458 | unhandled error | Stessi pattern in `TestUserIDFromContext`: `ks, _ :=`, `signerData, _ :=`, `signer, _, _ :=`, `blob, _ :=` |
| `internal/auth/auth_test.go` | ~490–491 | unhandled error | `ks.Lookup(fpRsa)` e `ks.List()` nel goroutine di concorrenza — return value completamente ignorati (nemmeno `_`) |

> **Nota sui test**: gli errori ignorati con `_` nei test andrebbero convertiti in chiamate a helper (es. `mustNewKeyStore(t, dir)`) che usano `t.Fatalf` in caso di errore, eliminando sia il warning che la possibilità di silenzio su fallimenti inattesi.

---

## Scope

### In scope

1. **Eliminazione di `internal/errors`**: il package non è importato da nessun package interno. Va rimosso completamente.

2. **`storage.sanitizeID` — regex compilata ad ogni chiamata**: spostare il `regexp.MustCompile` a variabile di package (`var sanitizeRE = regexp.MustCompile(...)`).

3. **Unhandled errors in `storage.writeProject`**:
   - `os.Remove(tmpPath)` nel defer: ignorare esplicitamente con `_ =` e commento (cleanup best-effort).
   - `tmp.Close()` nei percorsi di errore (righe 120, 124): usare `_ = tmp.Close()` con commento, oppure gestire l'errore.

4. **`cipher.NewGCM` — errore ignorato in `encryption/cipher.go`** (righe 25 e 57): convertire in `if gcm, err := cipher.NewGCM(block); err != nil { panic(...) }` con commento che spiega perché non può mai fallire per un AES block cipher standard.

5. **`io.ReadFull` — errore ignorato in `encryption/derive.go`** (riga 41): convertire in `if _, err := io.ReadFull(r, key); err != nil { panic(...) }` con commento che spiega il motivo.

6. **`buf.Write` — return value non catturato in `encryption/derive.go`** (riga 68): aggiungere `_ =` oppure ignorare in modo esplicito documentato (anche se `bytes.Buffer.Write` non fallisce mai).

7. **`fmt.Sprintf("%s", err)` ridondante in `auth/middleware.go`** (righe 65, 71): sostituire con `err.Error()`.

8. **Errori `json.Encode` ignorati — documentare l'intenzionalità**: in `internal/server/helpers.go:12` e `internal/auth/middleware.go:101` aggiungere commento esplicativo sul perché l'errore viene ignorato (il body è già committed dopo `WriteHeader`).

9. **Warning nei test — errori ignorati con `_`**: in `internal/auth/auth_test.go` convertire i pattern `x, _ := f()` in chiamate a helper dedicati (`mustNewKeyStore`, `mustReadFile`, `mustParseSigner`, `mustSign`) che usano `t.Fatal` in caso di errore.

10. **Warning nei test — return value completamente ignorati**: in `TestConcurrency` (`auth_test.go`), le chiamate `ks.Lookup(fpRsa)` e `ks.List()` nel goroutine anonimo ignor ano i return value senza nemmeno `_`. Convertire in `_, _ = ks.Lookup(...)` e `_, _ = ks.List()` con commento, oppure rimuovere le chiamate duplicate (già testate negli altri goroutine con gestione dell'errore).

11. **Copertura `internal/storage`** — portare `writeProject` (63.6%) ≥ 90%: aggiungere test per i percorsi di errore `CreateTemp` e `Rename`.

12. **Copertura `internal/ssh`** — portare `ParsePrivateKey` (80%) e `Sign` (85.7%) ≥ 90%: aggiungere casi per PEM malformato, tipo PEM non supportato, e `Sign` con signer che fallisce.

13. **Copertura `internal/server`** — portare `registerSecretsRoutes` (85.9%) ≥ 90%: coprire il percorso 500 di `handleListSecrets`.

14. **Uniformità gestione errori nel server**: verificare che tutti i `500` siano loggati con `.Error(...)` e nessun `4xx` usi `.Error(...)`.

15. **Uniformità del logging**: verificare coerenza delle chiavi di log (`"user"`, `"project"`, `"key"`, `"error"`, `"count"`).

### Out of scope

- Refactoring della struttura dei package.
- Modifiche alle API HTTP o alle firme delle funzioni pubbliche.
- Aggiunta di nuove funzionalità.
- Miglioramento della copertura dei package `cmd/*` (non testabili senza infrastruttura).
- Aggiunta di `staticcheck` come dipendenza se non già installata (usare solo `go vet` se non disponibile).

---

## Soglie di copertura post-cleanup

| Package | Copertura attuale | Target |
|---------|------------------|--------|
| `internal/auth` | 93.5% | ≥ 90% ✅ (solo verifica) |
| `internal/config` | 95.0% | ≥ 90% ✅ (solo verifica) |
| `internal/encryption` | 95.1% | ≥ 90% ✅ (solo verifica) |
| `internal/logging` | 100% | ≥ 90% ✅ (solo verifica) |
| `internal/server` | 91.1% | ≥ 90% ✅ (miglioramento opzionale) |
| `internal/ssh` | 91.4% | ≥ 90% ✅ (solo verifica, migliorabile) |
| `internal/storage` | 91.5% | ≥ 90% ✅ (miglioramento `writeProject`) |
| `internal/errors` | — | eliminato |


