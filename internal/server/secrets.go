package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"ekvs/internal/auth"
	"ekvs/internal/logging"
	"ekvs/internal/storage"
)

// secretEntry is the JSON representation of a single secret.
type secretEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// SecretsHandler returns an http.Handler that exposes secret management
// endpoints for a given project. It must be wrapped by auth.AuthMiddleware
// before mounting.
//
// Routes:
//
//	GET    /projects/{name}                → project info + list keys
//	GET    /projects/{name}/secrets        → list secrets with values
//	PUT    /projects/{name}/secrets/{key}  → set secret
//	GET    /projects/{name}/secrets/{key}  → get secret
//	DELETE /projects/{name}/secrets/{key}  → delete secret
func SecretsHandler(store *storage.Store, log logging.Logger) http.Handler {
	mux := http.NewServeMux()
	registerSecretsRoutes(mux, store, log)
	return mux
}

// registerSecretsRoutes registers secret management routes on mux.
func registerSecretsRoutes(mux *http.ServeMux, store *storage.Store, log logging.Logger) {
	// GET /projects/{name} — project metadata + key list
	mux.HandleFunc("GET /projects/{name}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleGetProject: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")
		keys, err := store.ListSecrets(userID, name)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			default:
				log.Error("handleGetProject: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}
		if keys == nil {
			keys = []string{}
		}

		log.Debug("project fetched", "user", userID, "project", name)
		writeJSON(w, http.StatusOK, map[string]any{"name": name, "keys": keys})
	})

	// GET /projects/{name}/secrets — full list with values
	mux.HandleFunc("GET /projects/{name}/secrets", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleListSecrets: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")
		keys, err := store.ListSecrets(userID, name)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			default:
				log.Error("handleListSecrets: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		secrets := make([]secretEntry, 0, len(keys))
		for _, k := range keys {
			val, err := store.GetSecret(userID, name, k)
			if err != nil {
				log.Error("handleListSecrets: GetSecret error", "key", k, "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
				return
			}
			secrets = append(secrets, secretEntry{Key: k, Value: val})
		}

		log.Debug("secrets listed", "user", userID, "project", name, "count", len(secrets))
		writeJSON(w, http.StatusOK, map[string]any{"secrets": secrets})
	})

	// PUT /projects/{name}/secrets/{key} — set secret
	mux.HandleFunc("PUT /projects/{name}/secrets/{key}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleSetSecret: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")
		key := r.PathValue("key")

		var body struct {
			Value string `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if body.Value == "" {
			writeError(w, http.StatusBadRequest, "value must not be empty")
			return
		}

		if err := store.SetSecret(userID, name, key, body.Value); err != nil {
			switch {
			case errors.Is(err, storage.ErrInvalidName):
				writeError(w, http.StatusBadRequest, "invalid key name")
			case errors.Is(err, storage.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			default:
				log.Error("handleSetSecret: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		log.Debug("secret set", "user", userID, "project", name, "key", key)
		writeJSON(w, http.StatusOK, map[string]string{"key": key})
	})

	// GET /projects/{name}/secrets/{key} — get single secret
	mux.HandleFunc("GET /projects/{name}/secrets/{key}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleGetSecret: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")
		key := r.PathValue("key")

		val, err := store.GetSecret(userID, name, key)
		if err != nil {
			switch {
			case errors.Is(err, storage.ErrProjectNotFound), errors.Is(err, storage.ErrKeyNotFound):
				writeError(w, http.StatusNotFound, "not found")
			default:
				log.Error("handleGetSecret: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		log.Debug("secret fetched", "user", userID, "project", name, "key", key)
		writeJSON(w, http.StatusOK, secretEntry{Key: key, Value: val})
	})

	// DELETE /projects/{name}/secrets/{key} — delete secret
	mux.HandleFunc("DELETE /projects/{name}/secrets/{key}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleDeleteSecret: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")
		key := r.PathValue("key")

		if err := store.DeleteSecret(userID, name, key); err != nil {
			switch {
			case errors.Is(err, storage.ErrProjectNotFound), errors.Is(err, storage.ErrKeyNotFound):
				writeError(w, http.StatusNotFound, "not found")
			default:
				log.Error("handleDeleteSecret: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		log.Debug("secret deleted", "user", userID, "project", name, "key", key)
		w.WriteHeader(http.StatusNoContent)
	})
}
