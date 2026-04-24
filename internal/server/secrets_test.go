package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ekvs/internal/auth"
	"ekvs/internal/storage"
)

// ---- helpers ----

// seedProject creates a project and optionally populates it with secrets.
func seedProject(t *testing.T, st *storage.Store, userID, project string, secrets map[string]string) {
	t.Helper()
	if err := st.CreateProject(userID, project); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	for k, v := range secrets {
		if err := st.SetSecret(userID, project, k, v); err != nil {
			t.Fatalf("SetSecret(%q): %v", k, err)
		}
	}
}

func newSecretsHandler(t *testing.T) (http.Handler, *storage.Store) {
	t.Helper()
	st := newStore(t)
	return SecretsHandler(st, noopLogger{}), st
}

// requestWithUserAndBody builds a request with userID in context and a JSON body.
func requestWithUserAndBody(method, target, userID string, body []byte) *http.Request {
	r := httptest.NewRequest(method, target, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r.WithContext(auth.NewContextWithUserID(r.Context(), userID))
}

// ---- GET /projects/{name} ----

func TestHandleGetProject(t *testing.T) {
	tests := []struct {
		name       string
		project    string
		setup      func(*storage.Store)
		wantStatus int
		wantKeys   []string // nil means don't check body
	}{
		{
			name:    "project with secrets returns sorted keys",
			project: "p",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", map[string]string{"b": "2", "a": "1"})
			},
			wantStatus: http.StatusOK,
			wantKeys:   []string{"a", "b"},
		},
		{
			name:    "project with no secrets returns empty keys",
			project: "p",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", nil)
			},
			wantStatus: http.StatusOK,
			wantKeys:   []string{},
		},
		{
			name:       "project not found",
			project:    "missing",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, st := newSecretsHandler(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			w := do(h, requestWithUser("GET", "/projects/"+tc.project, "u1"))
			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
			if tc.wantKeys != nil {
				var resp struct {
					Name string   `json:"name"`
					Keys []string `json:"keys"`
				}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.Name != tc.project {
					t.Errorf("name: got %q, want %q", resp.Name, tc.project)
				}
				if len(resp.Keys) != len(tc.wantKeys) {
					t.Fatalf("keys len: got %d, want %d (%v)", len(resp.Keys), len(tc.wantKeys), resp.Keys)
				}
				for i, k := range tc.wantKeys {
					if resp.Keys[i] != k {
						t.Errorf("keys[%d]: got %q, want %q", i, resp.Keys[i], k)
					}
				}
			}
		})
	}
}

func TestHandleGetProject_MissingUserID(t *testing.T) {
	h, _ := newSecretsHandler(t)
	w := do(h, httptest.NewRequest("GET", "/projects/p", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- GET /projects/{name}/secrets ----

func TestHandleListSecrets(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*storage.Store)
		wantStatus  int
		wantSecrets []secretEntry
	}{
		{
			name: "project with secrets sorted alphabetically",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", map[string]string{"b": "v2", "a": "v1"})
			},
			wantStatus:  http.StatusOK,
			wantSecrets: []secretEntry{{"a", "v1"}, {"b", "v2"}},
		},
		{
			name: "project with no secrets returns empty array",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", nil)
			},
			wantStatus:  http.StatusOK,
			wantSecrets: []secretEntry{},
		},
		{
			name:        "project not found",
			wantStatus:  http.StatusNotFound,
			wantSecrets: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, st := newSecretsHandler(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			w := do(h, requestWithUser("GET", "/projects/p/secrets", "u1"))
			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
			if tc.wantSecrets != nil {
				var resp struct {
					Secrets []secretEntry `json:"secrets"`
				}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if len(resp.Secrets) != len(tc.wantSecrets) {
					t.Fatalf("secrets len: got %d, want %d", len(resp.Secrets), len(tc.wantSecrets))
				}
				for i, want := range tc.wantSecrets {
					got := resp.Secrets[i]
					if got.Key != want.Key || got.Value != want.Value {
						t.Errorf("secrets[%d]: got {%q,%q}, want {%q,%q}", i, got.Key, got.Value, want.Key, want.Value)
					}
				}
			}
		})
	}
}

func TestHandleListSecrets_MissingUserID(t *testing.T) {
	h, _ := newSecretsHandler(t)
	w := do(h, httptest.NewRequest("GET", "/projects/p/secrets", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// TestHandleListSecrets_FileCorrupted verifies that when the project file on
// disk contains invalid JSON the handler returns a non-200 error response.
// This exercises both the ListSecrets and GetSecret internal-error branches.
func TestHandleListSecrets_FileCorrupted(t *testing.T) {
	// Use an explicit directory so we can locate the project file.
	storeDir := t.TempDir()
	st, err := storage.New(storeDir)
	if err != nil {
		t.Fatal(err)
	}
	seedProject(t, st, "u1", "p", map[string]string{"key1": "val1"})

	// Overwrite the project file with invalid JSON to trigger a parse error.
	// sanitizeID("u1") == "u1" (all chars are safe).
	projectFile := filepath.Join(storeDir, "u1", "p.json")
	if err := os.WriteFile(projectFile, []byte("{invalid json"), 0600); err != nil {
		t.Fatal(err)
	}

	h := SecretsHandler(st, noopLogger{})
	w := do(h, requestWithUser("GET", "/projects/p/secrets", "u1"))
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for corrupted project file, got 200")
	}
}

// ---- PUT /projects/{name}/secrets/{key} ----

func TestHandleSetSecret(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		body       []byte
		setup      func(*storage.Store)
		wantStatus int
	}{
		{
			name: "create new secret",
			key:  "mykey",
			body: []byte(`{"value":"aGVsbG8="}`),
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "overwrite existing secret",
			key:  "mykey",
			body: []byte(`{"value":"bmV3dmFsdWU="}`),
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", map[string]string{"mykey": "old"})
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed JSON",
			key:        "k",
			body:       []byte(`not-json`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "value field absent",
			key:        "k",
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "value empty string",
			key:        "k",
			body:       []byte(`{"value":""}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid key name",
			key:        "bad%20name",
			body:       []byte(`{"value":"x"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "project not found",
			key:        "k",
			body:       []byte(`{"value":"x"}`),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, st := newSecretsHandler(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			req := requestWithUserAndBody("PUT", "/projects/p/secrets/"+tc.key, "u1", tc.body)
			w := do(h, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
			if tc.wantStatus == http.StatusOK {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp["key"] != tc.key {
					t.Errorf("key: got %q, want %q", resp["key"], tc.key)
				}
			}
		})
	}
}

func TestHandleSetSecret_MissingUserID(t *testing.T) {
	h, _ := newSecretsHandler(t)
	req := httptest.NewRequest("PUT", "/projects/p/secrets/k", bytes.NewBufferString(`{"value":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := do(h, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- GET /projects/{name}/secrets/{key} ----

func TestHandleGetSecret(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		setup      func(*storage.Store)
		wantStatus int
		wantValue  string
	}{
		{
			name: "existing secret",
			key:  "mykey",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", map[string]string{"mykey": "aGVsbG8="})
			},
			wantStatus: http.StatusOK,
			wantValue:  "aGVsbG8=",
		},
		{
			name: "key not found",
			key:  "missing",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", nil)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "project not found",
			key:        "k",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, st := newSecretsHandler(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			w := do(h, requestWithUser("GET", "/projects/p/secrets/"+tc.key, "u1"))
			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
			if tc.wantValue != "" {
				var resp secretEntry
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp.Key != tc.key {
					t.Errorf("key: got %q, want %q", resp.Key, tc.key)
				}
				if resp.Value != tc.wantValue {
					t.Errorf("value: got %q, want %q", resp.Value, tc.wantValue)
				}
			}
		})
	}
}

func TestHandleGetSecret_MissingUserID(t *testing.T) {
	h, _ := newSecretsHandler(t)
	w := do(h, httptest.NewRequest("GET", "/projects/p/secrets/k", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- DELETE /projects/{name}/secrets/{key} ----

func TestHandleDeleteSecret(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		setup      func(*storage.Store)
		wantStatus int
	}{
		{
			name: "delete existing secret",
			key:  "mykey",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", map[string]string{"mykey": "val"})
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "key not found",
			key:  "missing",
			setup: func(st *storage.Store) {
				seedProject(t, st, "u1", "p", nil)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "project not found",
			key:        "k",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, st := newSecretsHandler(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			w := do(h, requestWithUser("DELETE", "/projects/p/secrets/"+tc.key, "u1"))
			if w.Code != tc.wantStatus {
				t.Fatalf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
		})
	}
}

func TestHandleDeleteSecret_MissingUserID(t *testing.T) {
	h, _ := newSecretsHandler(t)
	w := do(h, httptest.NewRequest("DELETE", "/projects/p/secrets/k", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- NewHandler ----

func TestNewHandler(t *testing.T) {
	st := newStore(t)
	h := NewHandler(st, noopLogger{})
	if h == nil {
		t.Fatal("NewHandler returned nil")
	}

	// Verify it serves both projects and secrets routes.
	if err := st.CreateProject("u1", "proj"); err != nil {
		t.Fatal(err)
	}
	w := do(h, requestWithUser("GET", "/projects", "u1"))
	if w.Code != http.StatusOK {
		t.Errorf("projects route: got %d, want 200", w.Code)
	}
	w = do(h, requestWithUser("GET", "/projects/proj", "u1"))
	if w.Code != http.StatusOK {
		t.Errorf("secrets route: got %d, want 200", w.Code)
	}
}

// ---- End-to-end secrets ----

func TestSecretsEndToEnd(t *testing.T) {
	h, st := newSecretsHandler(t)
	seedProject(t, st, "u1", "proj", nil)

	// PUT two secrets
	for _, kv := range []struct{ k, v string }{{"alpha", "v1"}, {"beta", "v2"}} {
		body, _ := json.Marshal(map[string]string{"value": kv.v})
		w := do(h, requestWithUserAndBody("PUT", "/projects/proj/secrets/"+kv.k, "u1", body))
		if w.Code != http.StatusOK {
			t.Fatalf("set %q: got %d", kv.k, w.Code)
		}
	}

	// GET /projects/proj — expect keys list
	w := do(h, requestWithUser("GET", "/projects/proj", "u1"))
	if w.Code != http.StatusOK {
		t.Fatalf("get project: got %d", w.Code)
	}
	var proj struct {
		Name string   `json:"name"`
		Keys []string `json:"keys"`
	}
	_ = json.NewDecoder(w.Body).Decode(&proj)
	if proj.Name != "proj" || len(proj.Keys) != 2 {
		t.Fatalf("unexpected project: %+v", proj)
	}

	// GET /projects/proj/secrets — expect full secrets
	w = do(h, requestWithUser("GET", "/projects/proj/secrets", "u1"))
	if w.Code != http.StatusOK {
		t.Fatalf("list secrets: got %d", w.Code)
	}
	var list struct {
		Secrets []secretEntry `json:"secrets"`
	}
	_ = json.NewDecoder(w.Body).Decode(&list)
	if len(list.Secrets) != 2 || list.Secrets[0].Key != "alpha" || list.Secrets[1].Key != "beta" {
		t.Fatalf("unexpected secrets: %+v", list.Secrets)
	}

	// GET single secret
	w = do(h, requestWithUser("GET", "/projects/proj/secrets/alpha", "u1"))
	if w.Code != http.StatusOK {
		t.Fatalf("get secret: got %d", w.Code)
	}
	var entry secretEntry
	_ = json.NewDecoder(w.Body).Decode(&entry)
	if entry.Value != "v1" {
		t.Errorf("value: got %q, want v1", entry.Value)
	}

	// DELETE and verify gone
	w = do(h, requestWithUser("DELETE", "/projects/proj/secrets/alpha", "u1"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d", w.Code)
	}
	w = do(h, requestWithUser("GET", "/projects/proj/secrets/alpha", "u1"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("after delete: got %d, want 404", w.Code)
	}
}
