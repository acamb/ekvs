package config

import "os"

const (
	defaultServerAddr  = "127.0.0.1:8080"
	defaultStoragePath = "./data"
	defaultLogLevel    = "info"
)

// Config holds the application-wide configuration.
type Config struct {
	ServerAddr  string
	StoragePath string
	LogLevel    string
}

// Load reads configuration from environment variables, falling back to
// sensible defaults when a variable is not set.
func Load() (*Config, error) {
	return &Config{
		ServerAddr:  envOr("EKVS_SERVER_ADDR", defaultServerAddr),
		StoragePath: envOr("EKVS_STORAGE_PATH", defaultStoragePath),
		LogLevel:    envOr("EKVS_LOG_LEVEL", defaultLogLevel),
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
