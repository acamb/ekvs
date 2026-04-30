# requirements.md — cli_export_file

## Goal

Extend `ekvs export` so that a single secret can be written directly to a file:
`ekvs export <projectName> <keyName> --output /path/to/file`.
The file contains only the raw decrypted value — no `KEY=` prefix, no trailing newline.
Existing stdout behaviour (`KEY=value\n`) is unchanged when `--output` is not set.

---

## Scope

### In scope

- Add a `--output` string flag to `exportCmd` in `internal/cli/export.go`.
- Validate that `--output` is only accepted when exactly two positional args are supplied
  (`projectName` + `keyName`); supplying `--output` with only `projectName` is an error.
- When `--output` is set in the single-key path: write the raw decrypted value (no trailing
  newline) to the specified file path using `os.WriteFile`, with permission `0600`.
- All other paths (`export proj`, `export proj key` to stdout) remain unchanged.
- Unit tests added to `internal/cli/export_test.go` following the existing style.

### Out of scope

- Writing all secrets to a file (multi-key `--output` is not supported).
- Appending to an existing file (`>>` semantics).
- Creating missing intermediate directories.
- A `-o` shorthand flag.
- HTTPS / TLS support.

---

## Decisions

| # | Decision |
|---|----------|
| 1 | **Flag name and type** — `--output` is a `string` flag registered with `exportCmd.Flags()` (local, not persistent). Default value `""` means "write to stdout". |
| 2 | **File write mode and permissions** — `os.WriteFile(path, []byte(plaintext), 0600)`. No additional `os` operations are needed. |
| 3 | **Trailing newline** — The file contains only the raw decrypted value bytes. No `\n` is appended. This differs intentionally from the stdout `KEY=value\n` format. |
| 4 | **`--output` requires `keyName`** — If `--output` is non-empty and `len(args) != 2`, `RunE` returns `fmt.Errorf("--output requires a keyName argument")` before any network call. |
| 5 | **File overwrite behaviour** — An existing file at the target path is silently overwritten. `os.WriteFile` truncates on open; no `--force` flag is needed. |
| 6 | **Error on directory-not-found** — If the parent directory does not exist, `os.WriteFile` returns an error; `RunE` wraps it as `fmt.Errorf("write output file: %w", err)`. |
| 7 | **No new dependencies** — Only `os` from the standard library is needed, already available in the package. |

