# validation.md — tui_auth

## Acceptance criteria

The feature is considered complete when **all** of the following criteria are met.

---

## 1. Unit tests

```bash
make test
```

- `ok  ekvs/internal/ssh` — no regressions, new passphrase tests pass.
- `ok  ekvs/internal/tui/session` — coverage ≥ 90 %.
- `ok  ekvs/internal/tui/auth` — coverage ≥ 90 %.
- No regressions on any other package.

Or, targeted:

```bash
go test ./internal/ssh/... ./internal/tui/session/... ./internal/tui/auth/... -v -cover
```

---

## 2. Build

```bash
go build ./cmd/tui/...
```

Must complete with no errors or warnings.

---

## 3. Unencrypted key — silent auth

Prepare an unencrypted Ed25519 key:

```bash
ssh-keygen -t ed25519 -N "" -f /tmp/test_id_ed25519
```

Create a config with that identity file:

```bash
cat > /tmp/test-ekvs-tui.yaml <<EOF
profiles:
  - name:          "test"
    server_url:    "http://127.0.0.1:9090"
    identity_file: "/tmp/test_id_ed25519"
    theme:         "default"
EOF
```

Run the TUI:

```bash
go run ./cmd/tui --config /tmp/test-ekvs-tui.yaml
```

1. The main menu is shown immediately (no passphrase screen).
2. Select `[A] Authenticate` (temporary menu item).
3. **Expected**: no passphrase prompt is shown; a brief "Authenticated" status
   message appears and the session is established.

---

## 4. Passphrase-protected key — prompt shown once

Prepare a passphrase-protected key:

```bash
ssh-keygen -t ed25519 -N "hunter2" -f /tmp/test_id_passphrase
```

Update the config:

```bash
cat > /tmp/test-ekvs-tui.yaml <<EOF
profiles:
  - name:          "test"
    server_url:    "http://127.0.0.1:9090"
    identity_file: "/tmp/test_id_passphrase"
    theme:         "default"
EOF
```

Run the TUI and trigger `[A] Authenticate`:

| Step | Action | Expected |
|------|--------|----------|
| 1 | Select `[A] Authenticate` | Passphrase input screen appears |
| 2 | Type `hunter2` and press `Enter` | Screen disappears; session established; main menu shown |
| 3 | Select `[A] Authenticate` again | **No prompt shown** — session already active |

---

## 5. Wrong passphrase — error and retry

Using the passphrase-protected key from criterion 4:

| Step | Action | Expected |
|------|--------|----------|
| 1 | Select `[A] Authenticate` | Passphrase input screen appears |
| 2 | Type `wrongpass` and press `Enter` | Error message shown: "incorrect passphrase" or similar |
| 3 | Press any key | Input is cleared; passphrase prompt re-appears |
| 4 | Type `hunter2` and press `Enter` | Session established; main menu shown |

---

## 6. Cancel auth

Using the passphrase-protected key:

| Step | Action | Expected |
|------|--------|----------|
| 1 | Select `[A] Authenticate` | Passphrase input screen appears |
| 2 | Press `Esc` | Main menu shown; session **not** established |

---

## 7. `SignRequest` correctness (unit tests)

Verified by unit tests in `internal/tui/auth/sign_test.go`:

- `X-Timestamp` equals `strconv.FormatInt(now.UTC().Unix(), 10)`.
- `X-Fingerprint` equals `ssh.FingerprintSHA256(publicKey)` (canonical `SHA256:<base64>` format).
- `X-Signature` is base64 **standard** encoding (`base64.StdEncoding`) — matching the server's
  `base64.StdEncoding.DecodeString` call in `internal/auth/middleware.go`.
- The decoded signature blob verifies correctly via
  `internal/ssh.Verify(sess.PublicKey, []byte(ssh.CanonicalRequest(method, path, now)), sigBlob)`,
  the same verification path used by `AuthMiddleware`.
- Calling `SignRequest` on an unauthenticated session returns `ErrNotAuthenticated`.

---

## 8. `ErrPassphraseRequired` sentinel

Verified by unit tests in `internal/ssh/ssh_test.go`:

- Calling `internal/ssh.ParsePrivateKey` on a passphrase-protected PEM returns
  `ErrPassphraseRequired` (checked via `errors.Is`).
- Calling `internal/ssh.ParsePrivateKeyWithPassphrase` with the correct passphrase
  returns a `crypto.Signer` whose public key fingerprint matches the key file.
- The returned `crypto.Signer` can be passed directly to `internal/ssh.Sign` and
  the resulting blob verified with `internal/ssh.Verify`.
- The returned `crypto.Signer` is accepted by `internal/encryption.DeriveKey`
  without error (smoke test for future `tui_encryption` milestone).


