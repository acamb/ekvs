# plan.md — tui_auth

Ordered list of implementation tasks.

---

## 1. Extend `internal/ssh` — passphrase-protected key parsing

**Files**: `internal/ssh/keys.go`, `internal/ssh/errors.go`

Add `ErrPassphraseRequired` sentinel to `errors.go`.

Update `ParsePrivateKey` in `keys.go` to detect when a key is passphrase-protected:
- Call `gossh.ParseRawPrivateKey(pemBytes)`.
- If the error is a `*gossh.PassphraseMissingError` (checked via `errors.As`),
  return `nil, nil, ErrPassphraseRequired`.

Add `ParsePrivateKeyWithPassphrase(pemBytes, passphrase []byte) (crypto.Signer, gossh.PublicKey, error)`:
- Calls `gossh.ParseRawPrivateKeyWithPassphrase(pemBytes, passphrase)`.
- Applies the same key-type allowlist as `ParsePrivateKey`.
- Returns `ErrUnsupportedKeyType` for unsupported types, or a wrapped error for
  wrong passphrase.

Update `internal/ssh/ssh_test.go` with table-driven tests covering:
- Unencrypted key (RSA, ECDSA, Ed25519) via `ParsePrivateKey` — success.
- Passphrase-protected key via `ParsePrivateKey` — returns `ErrPassphraseRequired`.
- Passphrase-protected key via `ParsePrivateKeyWithPassphrase` — correct passphrase succeeds.
- Passphrase-protected key via `ParsePrivateKeyWithPassphrase` — wrong passphrase returns error.
- Round-trip: sign with returned `crypto.Signer`, verify with `Verify(pub, msg, sig)`.

---

## 2. Create `internal/tui/session` package

**Files**: `session.go`, `session_test.go`

- Define `Session` struct (fields: `Signer crypto.Signer`, `PublicKey gossh.PublicKey`, `Fingerprint string`).
- `IsAuthenticated() bool`
- `Clear()`
- Unit tests: zero value is unauthenticated, `Clear` resets a populated session.

---

## 3. Create `internal/tui/auth` package — `SignRequest` helper

**File**: `internal/tui/auth/sign.go`, `sign_test.go`

`SignRequest(sess *session.Session, method, path string, now time.Time) (map[string]string, error)`

Implementation:
1. Return `ErrNotAuthenticated` if `sess.Signer == nil`.
2. Build canonical message: `ssh.CanonicalRequest(method, path, now)` — same
   function used by the server to reconstruct the message.
3. Sign: `ssh.Sign(sess.Signer, []byte(canonical))` — returns `gossh.Marshal`-ed blob.
4. Return map:
   - `"X-Timestamp"` → `strconv.FormatInt(now.UTC().Unix(), 10)`
   - `"X-Fingerprint"` → `sess.Fingerprint`
   - `"X-Signature"` → `base64.StdEncoding.EncodeToString(sigBlob)`

**Note**: `base64.StdEncoding` is required — the server middleware decodes with
`base64.StdEncoding.DecodeString` (see `internal/auth/middleware.go`).

Table-driven tests:
- Valid session: headers produced correctly; signature verifies via
  `ssh.Verify(sess.PublicKey, []byte(ssh.CanonicalRequest(method, path, now)), decoded_sig_blob)`.
- Unauthenticated session: returns `ErrNotAuthenticated`.
- Timestamp in headers matches `now.UTC().Unix()`.

---

## 4. Create `internal/tui/auth` package — passphrase input model

**File**: `internal/tui/auth/auth.go`, `auth_test.go`

bubbletea v2 model `Model` with states:
- `statePrompt` — text input for passphrase (masked with `*`).
- `stateError` — shows error message + "press any key to retry".

Messages:
- `AuthSuccessMsg{ Signer, PublicKey, Fingerprint }` — emitted on success.
- `AuthCancelMsg{}` — emitted when user presses `Esc` or `q`.

Behaviour:
1. On `Init`, attempt `ssh.ParsePrivateKey(pemBytes)`:
   - success → emit `AuthSuccessMsg` immediately (no prompt needed).
   - `ErrPassphraseRequired` → transition to `statePrompt`.
   - other error → transition to `stateError` with message.
2. In `statePrompt`, `Enter` submits passphrase:
   - success → emit `AuthSuccessMsg`.
   - failure → transition to `stateError`.
3. `Esc` / `q` in any state → emit `AuthCancelMsg`.

Styling uses the `theme.Theme` passed at construction.

Unit tests cover state transitions using `tea.NewTestModel`.

---

## 5. Integrate auth flow into root model

**File**: `internal/tui/root/root.go`

- Add `screenAuth` constant.
- Add `authModel auth.Model` and `session session.Session` fields to `Model`.
- Add `pendingScreen screen` field to remember which screen to return to after auth.
- Export a `TriggerAuth(returnTo screen)` helper (internal to package) that
  instantiates `auth.Model` with the identity file from the active profile and
  switches to `screenAuth`.
- Handle `auth.AuthSuccessMsg` in `Update`: populate `m.session`, switch to
  `pendingScreen`.
- Handle `auth.AuthCancelMsg` in `Update`: switch back to `screenMain` (auth
  cancelled, no session established).
- For now, add a temporary `[A] Authenticate` menu item to `mainModel` so the
  auth flow can be triggered manually for testing (will be removed or replaced
  in later milestones).

---

## 6. Unit tests — integration within root package

**File**: `internal/tui/root/root_test.go` (update existing or add cases)

- Test that dispatching the auth trigger from the main menu transitions to
  `screenAuth`.
- Test that `AuthSuccessMsg` populates the session and returns to `screenMain`.
- Test that `AuthCancelMsg` returns to `screenMain` without a session.

---

## 7. Validate

Run:
```bash
make test
go build ./cmd/tui/...
```

All tests pass, coverage ≥ 90 % on new packages, no regressions.



