package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultServerAddr  = "127.0.0.1:8080"
	defaultStoragePath = "./data"
	defaultKeysDir     = "./data/.keys"
	defaultLogLevel    = "info"
)

// Config holds the application-wide configuration.
type Config struct {
	ServerAddr  string
	StoragePath string
	KeysDir     string
	LogLevel    string
}

// yamlConfig mirrors Config with YAML struct tags.
type yamlConfig struct {
	ServerAddr  string `yaml:"server_addr"`
	StoragePath string `yaml:"storage_path"`
	KeysDir     string `yaml:"keys_dir"`
	LogLevel    string `yaml:"log_level"`
}

// Load reads configuration from environment variables, falling back to
// sensible defaults when a variable is not set.
func Load() (*Config, error) {
	return &Config{
		ServerAddr:  envOr("EKVS_SERVER_ADDR", defaultServerAddr),
		StoragePath: envOr("EKVS_STORAGE_PATH", defaultStoragePath),
		KeysDir:     envOr("EKVS_KEYS_DIR", defaultKeysDir),
		LogLevel:    envOr("EKVS_LOG_LEVEL", defaultLogLevel),
	}, nil
}

// LoadFromFile loads configuration from a YAML file at path, then applies
// environment variable overrides (env > file > default).
//
// If required is false and the file does not exist, the function falls back
// to Load() (defaults + env vars) without returning an error.
// If required is true and the file does not exist, an error is returned.
// A malformed YAML file always returns an error regardless of required.
func LoadFromFile(path string, required bool) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !required {
				return Load()
			}
			return nil, fmt.Errorf("config file %q not found", path)
		}
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var yc yamlConfig
	if err := yaml.Unmarshal(data, &yc); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	return &Config{
		ServerAddr:  coalesce("EKVS_SERVER_ADDR", yc.ServerAddr, defaultServerAddr),
		StoragePath: coalesce("EKVS_STORAGE_PATH", yc.StoragePath, defaultStoragePath),
		KeysDir:     coalesce("EKVS_KEYS_DIR", yc.KeysDir, defaultKeysDir),
		LogLevel:    coalesce("EKVS_LOG_LEVEL", yc.LogLevel, defaultLogLevel),
	}, nil
}

// coalesce returns the env var value if set, otherwise yamlVal if non-empty,
// otherwise fallback.
func coalesce(envKey, yamlVal, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if yamlVal != "" {
		return yamlVal
	}
	return fallback
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
