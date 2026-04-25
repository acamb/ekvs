# validation.md — tui_setup
## Acceptance criteria
The feature is considered complete when **all** of the following criteria are met.
---
## 1. Unit tests
```bash
make test
```
- Output: `ok  ekvs/internal/tui/config` and `ok  ekvs/internal/tui/theme` with no errors.
- Statement coverage ≥ 90% on `internal/tui/config` and `internal/tui/theme`.
- No regressions on other packages (`internal/auth`, `internal/config`, `internal/encryption`, `internal/ssh`, `internal/storage`, `internal/server`).
Or, targeted:
```bash
go test ./internal/tui/... -v -cover
```
---
## 2. Build
```bash
go build ./cmd/tui/...
```
Must complete with no errors or warnings.
---
## 3. Startup with a valid config — single profile
Create a configuration file with one profile:
```bash
cat > /tmp/test-ekvs-tui.yaml <<EOF
profiles:
  - name:          "test"
    server_url:    "http://127.0.0.1:9090"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "hacker"
EOF
```
Start the TUI:
```bash
go run ./cmd/tui --config /tmp/test-ekvs-tui.yaml
```
Verify:
- The profile selection screen is **not** shown (single profile → direct start).
- The TUI starts and shows the main menu with black background and green text (hacker theme).
- The four items `Projects`, `Secrets`, `Settings`, `Quit` are visible.
- The first item is highlighted with a visual cursor.
---
## 4. Menu navigation
With the TUI running (any theme):
| Action | Expected behaviour |
|--------|-------------------|
| `↓` or `j` | Cursor moves to the next item |
| `↑` or `k` | Cursor moves to the previous item |
| `↑` from the first item | Wrap-around: cursor moves to the last item |
| `↓` from the last item | Wrap-around: cursor moves to the first item |
| `Enter` on `Quit` | Application exits cleanly |
| `q` from any item | Application exits cleanly |
| `Ctrl+C` | Application exits cleanly |
| `Enter` on `Projects`, `Secrets`, `Settings` | No crash (placeholder behaviour) |
---
## 5. Adaptive theme
```bash
cat > /tmp/adaptive.yaml <<EOF
profiles:
  - name: "local"
    server_url: "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme: "adaptive"
EOF
go run ./cmd/tui --config /tmp/adaptive.yaml
```
The TUI must adapt to the terminal's colour scheme. Visual verification only.
---
## 6. Profile selection — multiple profiles
Create a file with two profiles:
```bash
cat > /tmp/multi.yaml <<EOF
profiles:
  - name:          "local"
    server_url:    "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme:         "adaptive"
  - name:          "production"
    server_url:    "https://ekvs.example.com"
    identity_file: "~/.ssh/id_rsa"
    theme:         "hacker"
EOF
go run ./cmd/tui --config /tmp/multi.yaml
```
Verify:
- The profile selection screen is shown **before** the main menu.
- Each row shows the profile name and its URL.
- Navigation with `↑↓` and selection with `Enter`.
- Selecting `local` → main menu with adaptive theme.
- Selecting `production` → main menu with hacker theme.
- Pressing `q`/`Ctrl+C` on the selection screen → application exits.
---
## 7. Duplicate names in the configuration file
```bash
cat > /tmp/dup.yaml <<EOF
profiles:
  - name: "local"
    server_url: "http://127.0.0.1:8080"
  - name: "local"
    server_url: "http://127.0.0.1:9090"
EOF
go run ./cmd/tui --config /tmp/dup.yaml
```
The program must exit with a fatal error indicating the duplicate name. Non-zero exit code.
---
## 8. Empty profiles list
```bash
cat > /tmp/empty.yaml <<EOF
profiles: []
EOF
go run ./cmd/tui --config /tmp/empty.yaml
```
The first-run wizard must be shown (same behaviour as an absent file).
---
## 9. Unknown theme
```bash
cat > /tmp/bad-theme.yaml <<EOF
profiles:
  - name: "test"
    server_url: "http://127.0.0.1:8080"
    identity_file: "~/.ssh/id_ed25519"
    theme: "invalid_theme"
EOF
go run ./cmd/tui --config /tmp/bad-theme.yaml
```
The program must exit with a fatal error indicating the unrecognised theme name. Non-zero exit code.
---
## 10. `--config` flag pointing to a missing file
```bash
go run ./cmd/tui --config /tmp/nonexistent-config.yaml
```
The program must exit with a fatal error and a descriptive message. Non-zero exit code.
---
## 11. First-run wizard — default config absent
Move to a temporary directory where `ekvs-tui.yaml` does not exist:
```bash
mkdir -p /tmp/ekvs-test-dir && cd /tmp/ekvs-test-dir
go run /home/andrea/src/ekvs/cmd/tui
```
Verify:
- The TUI does not start directly; the interactive wizard appears.
- The wizard shows an empty field for `name` (placeholder: `e.g. local`), then pre-filled fields for `server_url` and `identity_file`.
- After completing all fields, the user is asked whether to save the configuration.
---
## 12. Wizard — save configuration
Continuing from step 11:
- Answer `y` at the save prompt.
- Enter the file name (accept the default `ekvs-tui.yaml` by pressing Enter).
- Verify that `ekvs-tui.yaml` was created in the current directory with the entered values.
- Verify that the TUI starts normally after saving.
```bash
cat /tmp/ekvs-test-dir/ekvs-tui.yaml
```
The content must match the values entered in the wizard, in the `profiles:` structure.
---
## 13. Wizard — skip save
Repeat step 11, answering `n` at the save prompt:
- The TUI starts normally without creating any file.
- On the next run in the same directory, the wizard is shown again (no file was saved).
---
## 14. Malformed YAML
```bash
echo "this: is: not: valid: yaml: [" > /tmp/malformed.yaml
go run ./cmd/tui --config /tmp/malformed.yaml
```
The program must exit with a fatal error indicating the YAML parse problem. Non-zero exit code.
---
## 15. Example file
Verify that `ekvs-tui.yaml.example` exists in the repository root and contains at least two example profiles with all fields (`name`, `server_url`, `identity_file`, `theme`).
```bash
cat ekvs-tui.yaml.example
```
