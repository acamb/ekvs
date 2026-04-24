package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure env vars are unset for this test.
	os.Unsetenv("EKVS_SERVER_ADDR")
	os.Unsetenv("EKVS_STORAGE_PATH")
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
