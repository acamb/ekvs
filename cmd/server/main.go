package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"ekvs/internal/auth"
	"ekvs/internal/config"
	"ekvs/internal/logging"
	"ekvs/internal/server"
	"ekvs/internal/storage"
)

func main() {
	configPath := flag.String("config", "ekvs.yaml", "path to YAML config file")
	flag.Parse()

	// If the user explicitly passed --config, the file must exist (required=true).
	// If using the default path, a missing file is not an error.
	required := isFlagSet("config")
	cfg, err := config.LoadFromFile(*configPath, required)
	if err != nil {
		// Logger not available yet; write directly to stderr.
		os.Stderr.WriteString("ekvs: " + err.Error() + "\n")
		os.Exit(1)
	}

	log := logging.New(cfg.LogLevel)

	if err := ensureDirs(cfg, log); err != nil {
		os.Exit(1)
	}

	store, err := storage.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to open storage", "error", err)
		os.Exit(1)
	}

	ks, err := auth.NewKeyStore(cfg.KeysDir)
	if err != nil {
		log.Error("failed to open keystore", "error", err)
		os.Exit(1)
	}

	handler := server.ProjectsHandler(store, log)
	authed := auth.AuthMiddleware(ks, 30*time.Second, handler)

	log.Info("server starting", "addr", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, authed); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// ensureDirs creates StoragePath and KeysDir if they do not exist.
func ensureDirs(cfg *config.Config, log logging.Logger) error {
	for _, dir := range []string{cfg.StoragePath, cfg.KeysDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			log.Error("failed to create directory", "dir", dir, "error", err)
			return err
		}
	}
	return nil
}

// isFlagSet reports whether the named flag was explicitly set on the command line.
func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
