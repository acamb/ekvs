# validation.md — cli_exec

## Acceptance criteria

The feature is complete when **all** of the following criteria are met.

---

## 1. Unit tests

```bash
go test -race -count=1 -cover ./internal/cli/...
```

- All `TestExec_*` tests pass.
- Statement coverage for `exec.go` ≥ 85%.
- No regressions in any other package.

Targeted run:

```bash
go test -race -count=1 -v -run TestExec ./internal/cli/...
```

### Required tests

| Test name | What it verifies |
|-----------|-----------------|
| `TestExec_AllSecrets_HappyPath` | Two encrypted secrets returned by server; subprocess receives both `KEY=value` env entries. |
| `TestExec_SingleKey_HappyPath` | Server returns one encrypted secret for a named key; subprocess receives exactly that entry. |
| `TestExec_AllSecrets_Empty` | Server returns empty secrets list; subprocess runs normally without extra env vars. |
| `TestExec_ProjectNotFound` | `ListSecrets` returns 404; `RunE` returns error containing `"project not found"`. |
| `TestExec_KeyNotFound` | `GetSecret` returns 404; `RunE` returns error containing `"project or key not found"`. |
| `TestExec_ServerError` | Server returns 500; `RunE` returns a non-nil error. |
| `TestExec_DecryptFailure` | Blob encrypted with a different key; `session.Decrypt` fails; `RunE` returns decrypt error. |
| `TestExec_ProgramNotFound` | Program binary does not exist; `RunE` returns a non-nil error. |
| `TestExec_ProgramExitsNonZero` | Subprocess exits with code 1; `RunE` returns a non-nil `*exec.ExitError`. |
| `TestExec_ProgramReceivesArgs` | Args after `--` beyond the program name are forwarded to the subprocess. |
| `TestExec_InheritsParentEnv` | Subprocess env contains both a pre-existing parent var and the injected secret. |
| `TestExec_TooFewArgs` | Fewer than 2 positional args; cobra rejects before `RunE` is called. |

---

## 2. Build

```bash
go build ./...
```

Must complete with no errors.

---

## 3. Smoke test — inject all secrets

```bash
# Server running on :9090; project "demo" has secrets KEY1=val1, KEY2=val2
ekvs exec demo \
  --server 127.0.0.1:9090 \
  --identity ~/.ssh/id_ed25519 \
  -- env
```

Expected: output of `env` contains `KEY1=val1` and `KEY2=val2`.

---

## 4. Smoke test — inject a single secret

```bash
ekvs exec demo KEY1 \
  --server 127.0.0.1:9090 \
  --identity ~/.ssh/id_ed25519 \
  -- env
```

Expected: `KEY1=val1` appears; `KEY2` is not in the output.

---

## 5. Smoke test — error paths

```bash
# Non-existent project
ekvs exec no-such-project \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519 -- env
# Expected stderr: message containing "project not found"

# Non-existent key
ekvs exec demo NO_SUCH_KEY \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519 -- env
# Expected stderr: message containing "project or key not found"

# Non-existent binary
ekvs exec demo \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519 -- /no/such/binary
# Expected: exec error, non-zero exit
```

