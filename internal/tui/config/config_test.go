package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"ekvs/internal/tui/config"
)

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "ekvs-tui-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadFromFile(t *testing.T) {
	def := config.DefaultProfile()

	tests := []struct {
		name     string
		content  string // empty means use a non-existent path
		required bool
		wantNil  bool
		wantErr  bool
		check    func(t *testing.T, cf *config.ConfigFile)
	}{
		{
			name: "single complete profile",
			content: `profiles:
  - name: "local"
    server_url: "http://localhost:9090"
    identity_file: "~/.ssh/id_rsa"
    theme: "hacker"
`,
			check: func(t *testing.T, cf *config.ConfigFile) {
				if len(cf.Profiles) != 1 {
					t.Fatalf("expected 1 profile, got %d", len(cf.Profiles))
				}
				p := cf.Profiles[0]
				if p.Name != "local" || p.ServerURL != "http://localhost:9090" || p.Theme != "hacker" {
					t.Errorf("unexpected profile: %+v", p)
				}
			},
		},
		{
			name: "multiple profiles",
			content: `profiles:
  - name: "a"
    server_url: "http://a:8080"
    identity_file: "~/.ssh/a"
    theme: "adaptive"
  - name: "b"
    server_url: "http://b:8080"
    identity_file: "~/.ssh/b"
    theme: "hacker"
`,
			check: func(t *testing.T, cf *config.ConfigFile) {
				if len(cf.Profiles) != 2 {
					t.Fatalf("expected 2 profiles, got %d", len(cf.Profiles))
				}
			},
		},
		{
			name: "partial profile uses defaults",
			content: `profiles:
  - name: "name-only"
`,
			check: func(t *testing.T, cf *config.ConfigFile) {
				p := cf.Profiles[0]
				if p.ServerURL != def.ServerURL {
					t.Errorf("ServerURL: want %q, got %q", def.ServerURL, p.ServerURL)
				}
				if p.IdentityFile != def.IdentityFile {
					t.Errorf("IdentityFile: want %q, got %q", def.IdentityFile, p.IdentityFile)
				}
				if p.Theme != def.Theme {
					t.Errorf("Theme: want %q, got %q", def.Theme, p.Theme)
				}
			},
		},
		{
			name:    "missing file required=false returns nil nil",
			wantNil: true,
		},
		{
			name:     "missing file required=true returns error",
			required: true,
			wantErr:  true,
		},
		{
			name:    "malformed YAML returns error",
			content: "this: is: not: valid: yaml: [",
			wantErr: true,
		},
		{
			name:    "empty profiles list returns nil nil",
			content: "profiles: []\n",
			wantNil: true,
		},
		{
			name: "duplicate names return error",
			content: `profiles:
  - name: "dup"
    server_url: "http://a:8080"
  - name: "dup"
    server_url: "http://b:8080"
`,
			wantErr: true,
		},
		{
			name: "profile without name returns error",
			content: `profiles:
  - server_url: "http://a:8080"
`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var path string
			if tc.content == "" {
				path = filepath.Join(t.TempDir(), "nonexistent.yaml")
			} else {
				path = writeTemp(t, tc.content)
			}

			cf, err := config.LoadFromFile(path, tc.required)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.wantNil {
				if cf != nil {
					t.Fatalf("expected nil, got %+v", cf)
				}
				return
			}
			if cf == nil {
				t.Fatal("expected ConfigFile non nil")
			}
			if tc.check != nil {
				tc.check(t, cf)
			}
		})
	}
}

func TestSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ekvs-tui.yaml")

	original := &config.ConfigFile{
		Profiles: []config.Profile{
			{Name: "test", ServerURL: "http://x:1234", IdentityFile: "~/.ssh/x", Theme: "hacker"},
		},
	}

	if err := config.Save(path, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := config.LoadFromFile(path, true)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if loaded == nil || len(loaded.Profiles) != 1 {
		t.Fatal("missing profiles after round-trip")
	}
	p := loaded.Profiles[0]
	o := original.Profiles[0]
	if p.Name != o.Name || p.ServerURL != o.ServerURL || p.IdentityFile != o.IdentityFile || p.Theme != o.Theme {
		t.Errorf("profile differs after round-trip: got %+v, want %+v", p, o)
	}
}
