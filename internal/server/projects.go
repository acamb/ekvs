package server

import (
	"errors"
	"net/http"

	"ekvs/internal/auth"
	"ekvs/internal/logging"
	"ekvs/internal/storage"
)

// ProjectsHandler returns an http.Handler that exposes project management
// endpoints. It must be wrapped by auth.AuthMiddleware before mounting.
//
// Routes:
//
//	POST   /projects/{name}  → create project
//	GET    /projects         → list projects
//	DELETE /projects/{name}  → delete project
func ProjectsHandler(store *storage.Store, log logging.Logger) http.Handler {
	mux := http.NewServeMux()
	registerProjectsRoutes(mux, store, log)
	return mux
}

// registerProjectsRoutes registers project management routes on mux.
func registerProjectsRoutes(mux *http.ServeMux, store *storage.Store, log logging.Logger) {
	mux.HandleFunc("POST /projects/{name}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleCreateProject: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")

		if err := store.CreateProject(userID, name); err != nil {
			switch {
			case errors.Is(err, storage.ErrInvalidName):
				writeError(w, http.StatusBadRequest, "invalid project name")
			case errors.Is(err, storage.ErrProjectAlreadyExists):
				writeError(w, http.StatusConflict, "project already exists")
			default:
				log.Error("handleCreateProject: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		log.Debug("project created", "user", userID, "project", name)
		writeJSON(w, http.StatusCreated, map[string]string{"name": name})
	})

	mux.HandleFunc("GET /projects", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleListProjects: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		projects, err := store.ListProjects(userID)
		if err != nil {
			log.Error("handleListProjects: store error", "error", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		log.Debug("projects listed", "user", userID, "count", len(projects))
		writeJSON(w, http.StatusOK, map[string][]string{"projects": projects})
	})

	mux.HandleFunc("DELETE /projects/{name}", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			log.Error("handleDeleteProject: missing userID in context")
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		name := r.PathValue("name")

		if err := store.DeleteProject(userID, name); err != nil {
			switch {
			case errors.Is(err, storage.ErrProjectNotFound):
				writeError(w, http.StatusNotFound, "project not found")
			default:
				log.Error("handleDeleteProject: store error", "error", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
			return
		}

		log.Debug("project deleted", "user", userID, "project", name)
		w.WriteHeader(http.StatusNoContent)
	})
}
