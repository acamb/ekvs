package server

import (
	"encoding/json"
	"net/http"
)

// writeJSON sets Content-Type, writes status and encodes v as JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encoding errors are intentionally ignored: WriteHeader has already been
	// called, so the response status is committed and there is nothing useful
	// we can do if the body write fails.
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error body with the given HTTP status.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
