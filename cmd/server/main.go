package main

import (
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
	cfg, err := config.Load()
	if err != nil {
		// config.Load never actually errors currently, but be defensive.
		panic(err)
	}

	log := logging.New(cfg.LogLevel)

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
