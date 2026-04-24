# plan.md — server_auth

## Ordered Task List

1. **Create package skeleton**
   Create `internal/auth/` with:
   - `errors.go` — error sentinels
   - `keystore.go` — `KeyStore` struct, constructor, `Lookup`, `List`
   - `middleware.go` — `AuthMiddleware` and `UserIDFromContext`
   - `auth_test.go` — all unit tests

2. **Define error sentinels (`errors.go`)**
   Declare:
   - `ErrKeyNotFound`
   - `ErrInvalidSignature` — wraps `ssh.ErrInvalidSignature`
   - `ErrReplayDetected` — wraps `ssh.ErrReplayDetected`

3. **Implement `KeyStore` (`keystore.go`)**
   The server never writes to `.keys/`; it only reads.

   - `NewKeyStore(keysDir string) (*KeyStore, error)`:
     - Verify `keysDir` is accessible via `os.Stat`; return error if not.
     - Store `dir` and initialise `sync.RWMutex`.

   - `Lookup(fingerprint string) (gossh.PublicKey, error)`:
     - Acquire read lock.
     - `os.ReadDir(ks.dir)`, filter entries ending in `.pub`.
     - For each file: `os.ReadFile` → `ssh.ParseAuthorizedKey` (skip on parse error).
     - Compare `ssh.Fingerprint(pub)` with the requested fingerprint.
     - Return the first match or `ErrKeyNotFound` if none found.

   - `List() ([]string, error)`:
     - Acquire read lock.
     - Same scan as `Lookup` but collect all valid fingerprints.
     - Sort and return. Unparseable files are silently skipped.
     - Return empty non-nil slice if no valid keys exist.

4. **Implement `AuthMiddleware` and `UserIDFromContext` (`middleware.go`)**

   Unexported context key type:
   ```go
   type ctxKey struct{}
   var ctxKeyUserID = ctxKey{}
   ```

   `AuthMiddleware(ks *KeyStore, window time.Duration, next http.Handler) http.Handler`:
   1. Read `X-Timestamp`, `X-Fingerprint`, `X-Signature`; return 401 if any missing.
   2. Parse `X-Timestamp` as `int64` (`tsUnix`); convert to `time.Time` via `time.Unix(tsUnix, 0)`; return 401 if parse fails.
   3. `ks.Lookup(fingerprint)`; return 401 if `ErrKeyNotFound`.
   4. Reconstruct canonical message: `ssh.CanonicalRequest(r.Method, r.URL.Path, tsTime)`.
   5. `base64.StdEncoding.DecodeString(X-Signature)`; return 401 if malformed.
   6. `ssh.Verify(pub, []byte(message), sigBlob)`; return 401 on error.
   7. `ssh.CheckTimestamp(tsTime, window)`; return 401 on error. Standard production value: `30*time.Second`.
   8. `next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKeyUserID, fingerprint)))`.

   Helper for JSON error responses:
   ```go
   func writeError(w http.ResponseWriter, status int, msg string)
   ```

   `UserIDFromContext(ctx context.Context) (string, bool)`:
   - Return `ctx.Value(ctxKeyUserID).(string)`, ok.

5. **Write unit tests (`auth_test.go`)**
   Use `internal/ssh/testdata/` key files for real key material.

   Table-driven tests covering:
   - `NewKeyStore`: accessible dir → success; inaccessible dir → error.
   - `Lookup`: key present → matching public key; fingerprint absent → `ErrKeyNotFound`; dir with unparseable files → skipped, still finds valid key.
   - `List`: empty dir → empty slice; one valid key → one fingerprint; multiple keys → sorted; mix of valid and unparseable files → only valid ones returned.
   - `AuthMiddleware`: all valid → next called + context populated; missing `X-Timestamp` → 401; missing `X-Fingerprint` → 401; missing `X-Signature` → 401; bad timestamp format → 401; unknown fingerprint → 401; wrong signature → 401; expired timestamp → 401.
   - `UserIDFromContext`: populated context → fingerprint + true; empty context → "" + false.
   - Concurrency: N goroutines calling `Lookup` and `List` concurrently → no data races.

6. **Run `go mod tidy` and validate**
   Run `go mod tidy` (no new deps expected), then `make test`;
   confirm `ok  ekvs/internal/auth` with coverage ≥ 90%.

