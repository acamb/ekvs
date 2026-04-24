package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("EKVS_SERVER_ADDR")
	os.Unsetenv("EKVS_STORAGE_PATH")
	os.Unsetenv("EKVS_KEYS_DIR")
	os.Unsetenv("EKVS_LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"ServerAddr", cfg.ServerAddr, defaultServerAddr},
		{"StoragePath", cfg.StoragePath, defaultStoragePath},
		{"KeysDir", cfg.KeysDir, defaultKeysDir},
		{"LogLevel", cfg.LogLevel, defaultLogLevel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	tests := []struct {
		envKey string
		value  string
		check  func(*Config) string
	}{
		{"EKVS_SERVER_ADDR", "0.0.0.0:9090", func(c *Config) string { return c.ServerAddr }},
		{"EKVS_STORAGE_PATH", "/tmp/ekvs", func(c *Config) string { return c.StoragePath }},
		{"EKVS_KEYS_DIR", "/tmp/ekvs/.keys", func(c *Config) string { return c.KeysDir }},
		{"EKVS_LOG_LEVEL", "debug", func(c *Config) string { return c.LogLevel }},
	}
	for _, tc := range tests {
		t.Run(tc.envKey, func(t *testing.T) {
			t.Setenv(tc.envKey, tc.value)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() unexpected error: %v", err)
			}
			if got := tc.check(cfg); got != tc.value {
				t.Errorf("got %q, want %q", got, tc.value)
			}
		})
	}
}

// writeYAML writes content to a temp file and returns its path.
func writeYAML(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "ekvs.yaml")
	if err := os.WriteFile(f, []byte(content), 0600); err != nil {
		t.Fatalf("writeYAML: %v", err)
	}
	return f
}

func TestLoadFromFile_AllFields(t *testing.T) {
	os.Unsetenv("EKVS_SERVER_ADDR")
	os.Unsetenv("EKVS_STORAGE_PATH")
	os.Unsetenv("EKVS_KEYS_DIR")
	os.Unsetenv("EKVS_LOG_LEVEL")

	path := writeYAML(t, `
server_addr:  "0.0.0.0:9090"
storage_path: "/tmp/data"
keys_dir:     "/tmp/data/.keys"
log_level:    "debug"
`)
	cfg, err := LoadFromFile(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &Config{
		ServerAddr:  "0.0.0.0:9090",
		StoragePath: "/tmp/data",
		KeysDir:     "/tmp/data/.keys",
		LogLevel:    "debug",
	}
	if *cfg != *want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadFromFile_EnvOverridesFile(t *testing.T) {
	t.Setenv("EKVS_SERVER_ADDR", "127.0.0.1:7777")
	os.Unsetenv("EKVS_STORAGE_PATH")
	os.Unsetenv("EKVS_KEYS_DIR")
	os.Unsetenv("EKVS_LOG_LEVEL")

	path := writeYAML(t, `server_addr: "0.0.0.0:9090"`)
	cfg, err := LoadFromFile(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ServerAddr != "127.0.0.1:7777" {
		t.Errorf("env should win: got %q, want %q", cfg.ServerAddr, "127.0.0.1:7777")
	}
}

func TestLoadFromFile_PartialFields(t *testing.T) {
	os.Unsetenv("EKVS_SERVER_ADDR")
	os.Unsetenv("EKVS_STORAGE_PATH")
	os.Unsetenv("EKVS_KEYS_DIR")
	os.Unsetenv("EKVS_LOG_LEVEL")

	path := writeYAML(t, `log_level: "warn"`)
	cfg, err := LoadFromFile(path, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "warn" {
		t.Errorf("log_level: got %q, want %q", cfg.LogLevel, "warn")
	}
	if cfg.ServerAddr != defaultServerAddr {
		t.Errorf("server_addr: got %q, want default %q", cfg.ServerAddr, defaultServerAddr)
	}
	if cfg.StoragePath != defaultStoragePath {
		t.Errorf("storage_path: got %q, want default %q", cfg.StoragePath, defaultStoragePath)
	}
	if cfg.KeysDir != defaultKeysDir {
		t.Errorf("keys_dir: got %q, want default %q", cfg.KeysDir, defaultKeysDir)
	}
}

func TestLoadFromFile_MissingNotRequired(t *testing.T) {
	os.Unsetenv("EKVS_SERVER_ADDR")
	os.Unsetenv("EKVS_STORAGE_PATH")
	os.Unsetenv("EKVS_KEYS_DIR")
	os.Unsetenv("EKVS_LOG_LEVEL")

	cfg, err := LoadFromFile("/nonexistent/path/ekvs.yaml", false)
	if err != nil {
		t.Fatalf("missing+not-required should not error: %v", err)
	}
	if cfg.ServerAddr != defaultServerAddr {
		t.Errorf("got %q, want default %q", cfg.ServerAddr, defaultServerAddr)
	}
}

func TestLoadFromFile_MissingRequired(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/ekvs.yaml", true)
	if err == nil {
		t.Fatal("expected error for missing required file, got nil")
	}
}

func TestLoadFromFile_Malformed(t *testing.T) {
	path := writeYAML(t, `{invalid yaml:::`)
	_, err := LoadFromFile(path, true)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}
