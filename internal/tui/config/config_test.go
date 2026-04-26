package config_test

import (
	"os"
	"path/filepath"
	"strings"
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

// ── ExpandHome ────────────────────────────────────────────────────────────────

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory:", err)
	}

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, got string)
	}{
		{
			name:  "replaces leading tilde with home",
			input: "~/.ssh/id_ed25519",
			check: func(t *testing.T, got string) {
				if !strings.HasPrefix(got, home) {
					t.Errorf("expected prefix %q, got %q", home, got)
				}
				if strings.Contains(got, "~") {
					t.Errorf("result should not contain ~, got %q", got)
				}
			},
		},
		{
			name:  "absolute path unchanged",
			input: "/absolute/path",
			check: func(t *testing.T, got string) {
				if got != "/absolute/path" {
					t.Errorf("want %q, got %q", "/absolute/path", got)
				}
			},
		},
		{
			name:  "relative path without tilde unchanged",
			input: "relative/path",
			check: func(t *testing.T, got string) {
				if got != "relative/path" {
					t.Errorf("want %q, got %q", "relative/path", got)
				}
			},
		},
		{
			name:  "empty string unchanged",
			input: "",
			check: func(t *testing.T, got string) {
				if got != "" {
					t.Errorf("want empty string, got %q", got)
				}
			},
		},
		{
			name:  "tilde alone expands to home",
			input: "~",
			check: func(t *testing.T, got string) {
				if got != home {
					t.Errorf("want %q, got %q", home, got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := config.ExpandHome(tc.input)
			tc.check(t, got)
		})
	}
}

// ── SSHDir ────────────────────────────────────────────────────────────────────

func TestSSHDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory:", err)
	}

	got, err := config.SSHDir()
	if err != nil {
		t.Fatalf("SSHDir returned unexpected error: %v", err)
	}
	want := filepath.Join(home, ".ssh")
	if got != want {
		t.Errorf("SSHDir: want %q, got %q", want, got)
	}
}

// ── FindProfile ───────────────────────────────────────────────────────────────

func TestFindProfile(t *testing.T) {
	cf := &config.ConfigFile{
		Profiles: []config.Profile{
			{Name: "alpha", ServerURL: "http://a"},
			{Name: "beta", ServerURL: "http://b"},
		},
	}

	t.Run("found", func(t *testing.T) {
		p, idx, ok := cf.FindProfile("beta")
		if !ok || idx != 1 || p.Name != "beta" {
			t.Errorf("FindProfile(beta): got (%v, %d, %v)", p, idx, ok)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, idx, ok := cf.FindProfile("missing")
		if ok || idx != -1 {
			t.Errorf("FindProfile(missing): expected (-1, false), got (%d, %v)", idx, ok)
		}
	})
}

// ── UpsertProfile ─────────────────────────────────────────────────────────────

func TestUpsertProfile(t *testing.T) {
	t.Run("adds new unique profile", func(t *testing.T) {
		cf := &config.ConfigFile{}
		p := config.Profile{Name: "new", ServerURL: "http://new"}
		if err := cf.UpsertProfile(p); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cf.Profiles) != 1 || cf.Profiles[0].Name != "new" {
			t.Errorf("profile not added: %+v", cf.Profiles)
		}
	})

	t.Run("replaces existing profile in-place", func(t *testing.T) {
		cf := &config.ConfigFile{
			Profiles: []config.Profile{
				{Name: "existing", ServerURL: "http://old"},
				{Name: "other", ServerURL: "http://other"},
			},
		}
		updated := config.Profile{Name: "existing", ServerURL: "http://new"}
		if err := cf.UpsertProfile(updated); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cf.Profiles) != 2 {
			t.Fatalf("extra profiles added: want 2, got %d", len(cf.Profiles))
		}
		if cf.Profiles[0].ServerURL != "http://new" {
			t.Errorf("profile not replaced: %+v", cf.Profiles[0])
		}
		// Second profile must be untouched.
		if cf.Profiles[1].Name != "other" {
			t.Errorf("other profile was moved: %+v", cf.Profiles[1])
		}
	})

	t.Run("error on empty name", func(t *testing.T) {
		cf := &config.ConfigFile{}
		if err := cf.UpsertProfile(config.Profile{Name: ""}); err == nil {
			t.Error("expected error for empty name, got nil")
		}
	})
}

// ── UpdateProfile ─────────────────────────────────────────────────────────────

func TestUpdateProfile(t *testing.T) {
	base := func() *config.ConfigFile {
		return &config.ConfigFile{
			Profiles: []config.Profile{
				{Name: "alpha", ServerURL: "http://a", Theme: "adaptive"},
				{Name: "beta", ServerURL: "http://b", Theme: "hacker"},
			},
		}
	}

	t.Run("renames profile to new unique name", func(t *testing.T) {
		cf := base()
		updated := config.Profile{Name: "gamma", ServerURL: "http://a", Theme: "adaptive"}
		if err := cf.UpdateProfile("alpha", updated); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cf.Profiles[0].Name != "gamma" {
			t.Errorf("want name=gamma, got %q", cf.Profiles[0].Name)
		}
	})

	t.Run("keeps same name (no-rename edit)", func(t *testing.T) {
		cf := base()
		updated := config.Profile{Name: "alpha", ServerURL: "http://updated", Theme: "hacker"}
		if err := cf.UpdateProfile("alpha", updated); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cf.Profiles[0].ServerURL != "http://updated" {
			t.Errorf("ServerURL not updated: %q", cf.Profiles[0].ServerURL)
		}
	})

	t.Run("error when new name conflicts with another profile", func(t *testing.T) {
		cf := base()
		conflict := config.Profile{Name: "beta", ServerURL: "http://a", Theme: "adaptive"}
		if err := cf.UpdateProfile("alpha", conflict); err == nil {
			t.Error("expected conflict error, got nil")
		}
	})

	t.Run("error when oldName not found", func(t *testing.T) {
		cf := base()
		if err := cf.UpdateProfile("missing", config.Profile{Name: "x"}); err == nil {
			t.Error("expected not-found error, got nil")
		}
	})

	t.Run("error on empty new name", func(t *testing.T) {
		cf := base()
		if err := cf.UpdateProfile("alpha", config.Profile{Name: ""}); err == nil {
			t.Error("expected error for empty name, got nil")
		}
	})
}

// ── DeleteProfile ─────────────────────────────────────────────────────────────

func TestDeleteProfile(t *testing.T) {
	t.Run("removes existing profile", func(t *testing.T) {
		cf := &config.ConfigFile{
			Profiles: []config.Profile{
				{Name: "a"},
				{Name: "b"},
				{Name: "c"},
			},
		}
		if err := cf.DeleteProfile("b"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cf.Profiles) != 2 {
			t.Fatalf("want 2 profiles, got %d", len(cf.Profiles))
		}
		for _, p := range cf.Profiles {
			if p.Name == "b" {
				t.Error("deleted profile still present")
			}
		}
	})

	t.Run("error when name not found", func(t *testing.T) {
		cf := &config.ConfigFile{Profiles: []config.Profile{{Name: "a"}}}
		if err := cf.DeleteProfile("missing"); err == nil {
			t.Error("expected error, got nil")
		}
	})
}

// ── CRUD save/load round-trips ────────────────────────────────────────────────

func TestCRUD_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ekvs-tui.yaml")

	cf := &config.ConfigFile{}

	// Create.
	if err := cf.UpsertProfile(config.Profile{Name: "p1", ServerURL: "http://p1", Theme: "adaptive"}); err != nil {
		t.Fatalf("UpsertProfile: %v", err)
	}
	if err := config.Save(path, cf); err != nil {
		t.Fatalf("Save after create: %v", err)
	}

	// Edit (rename).
	if err := cf.UpdateProfile("p1", config.Profile{Name: "p1-renamed", ServerURL: "http://p1", Theme: "hacker"}); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if err := config.Save(path, cf); err != nil {
		t.Fatalf("Save after edit: %v", err)
	}

	// Delete.
	if err := cf.DeleteProfile("p1-renamed"); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}
	if err := config.Save(path, cf); err != nil {
		t.Fatalf("Save after delete: %v", err)
	}

	// The config on disk should now have no profiles (empty profiles list).
	loaded, err := config.LoadFromFile(path, false)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	// LoadFromFile returns nil when profiles list is empty.
	if loaded != nil {
		t.Errorf("expected nil after deleting all profiles, got %+v", loaded)
	}
}
