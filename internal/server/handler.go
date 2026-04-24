package server

import (
	"net/http"

	"ekvs/internal/logging"
	"ekvs/internal/storage"
)

// NewHandler returns an http.Handler that combines all project and secret
// management routes on a single ServeMux. It must be wrapped by
// auth.AuthMiddleware before mounting.
func NewHandler(store *storage.Store, log logging.Logger) http.Handler {
	mux := http.NewServeMux()
	registerProjectsRoutes(mux, store, log)
	registerSecretsRoutes(mux, store, log)
	return mux
}
