# validation.md — cli_export_file

## Acceptance criteria

The feature is complete when **all** of the following criteria are met.

---

## 1. Unit tests

```bash
go test -count=1 -cover ./internal/cli/...
```

- All `TestExport_OutputFile_*` tests pass.
- All pre-existing `TestExport_*` tests continue to pass (no regressions).
- Statement coverage for `export.go` ≥ 85%.

Targeted run:

```bash
go test -count=1 -v -run TestExport_OutputFile ./internal/cli/...
```

### Required tests

| Test name | What it verifies |
|-----------|-----------------|
| `TestExport_OutputFile_HappyPath` | Server returns one encrypted blob; file is created at `--output` path; content equals raw plaintext with no `KEY=` prefix and no trailing newline. |
| `TestExport_OutputFile_Overwrite` | Output file pre-exists with different content; after export the file contains the new plaintext only (old content fully replaced). |
| `TestExport_OutputFile_MissingKeyName` | `--output` supplied with only `projectName` (1 positional arg); `RunE` returns error containing `"--output requires a keyName argument"`; no HTTP request is made. |
| `TestExport_OutputFile_DirectoryNotFound` | `--output` path points into a non-existent directory; `RunE` returns error containing `"write output file"`; no output file is created. |
| `TestExport_OutputFile_DecryptFailure` | Server returns a blob encrypted with a different key; `session.Decrypt` fails; `RunE` returns non-nil error; output file is **not** created. |

---

## 2. Build

```bash
go build ./...
```

Must complete with no errors.

---

## 3. Smoke test — write single secret to file

```bash
# Server running on :9090; project "demo" has secret DB_PASS=hunter2
ekvs export demo DB_PASS \
  --server 127.0.0.1:9090 \
  --identity ~/.ssh/id_ed25519 \
  --output /tmp/db_pass.txt

cat /tmp/db_pass.txt
```

Expected: `hunter2` with no trailing newline and no `DB_PASS=` prefix.

---

## 4. Smoke test — error paths

```bash
# --output without keyName
ekvs export demo \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519 \
  --output /tmp/out.txt
# Expected stderr: "--output requires a keyName argument", non-zero exit

# --output pointing to a non-existent directory
ekvs export demo DB_PASS \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519 \
  --output /no/such/dir/out.txt
# Expected stderr: message containing "write output file", non-zero exit
```

---

## 5. Regression check — stdout paths unaffected

```bash
# All secrets still print KEY=value\n to stdout
ekvs export demo \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519
# Expected: DB_PASS=hunter2 (and all other secrets) on stdout

# Single key without --output still prints KEY=value\n
ekvs export demo DB_PASS \
  --server 127.0.0.1:9090 --identity ~/.ssh/id_ed25519
# Expected: DB_PASS=hunter2\n on stdout
```

