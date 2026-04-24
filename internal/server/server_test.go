package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"ekvs/internal/auth"
	"ekvs/internal/logging"
	"ekvs/internal/storage"
)

// noopLogger satisfies logging.Logger and discards all output.
type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}

var _ logging.Logger = noopLogger{}

// newStore returns a *storage.Store rooted at a fresh temp dir.
func newStore(t *testing.T) *storage.Store {
	t.Helper()
	st, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	return st
}

// requestWithUser builds an httptest.Request with userID injected in context.
func requestWithUser(method, target, userID string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	return r.WithContext(auth.NewContextWithUserID(r.Context(), userID))
}

// do runs req through h and returns the response recorder.
func do(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// ---- CreateProject ----

func TestHandleCreateProject(t *testing.T) {
	tests := []struct {
		name       string
		project    string
		setup      func(*storage.Store)
		wantStatus int
		wantName   string
	}{
		{
			name:       "valid project",
			project:    "alpha",
			wantStatus: http.StatusCreated,
			wantName:   "alpha",
		},
		{
			name:       "invalid name",
			project:    "!!bad!!",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "already exists",
			project: "alpha",
			setup: func(st *storage.Store) {
				_ = st.CreateProject("user1", "alpha")
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := newStore(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			h := ProjectsHandler(st, noopLogger{})
			w := do(h, requestWithUser("POST", "/projects/"+tc.project, "user1"))

			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
			if tc.wantName != "" {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if resp["name"] != tc.wantName {
					t.Errorf("name: got %q, want %q", resp["name"], tc.wantName)
				}
			}
		})
	}
}

func TestHandleCreateProject_MissingUserID(t *testing.T) {
	h := ProjectsHandler(newStore(t), noopLogger{})
	w := do(h, httptest.NewRequest("POST", "/projects/alpha", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- ListProjects ----

func TestHandleListProjects(t *testing.T) {
	t.Run("no projects", func(t *testing.T) {
		h := ProjectsHandler(newStore(t), noopLogger{})
		w := do(h, requestWithUser("GET", "/projects", "user1"))

		if w.Code != http.StatusOK {
			t.Errorf("got %d, want 200", w.Code)
		}
		var resp map[string][]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp["projects"]) != 0 {
			t.Errorf("expected empty list, got %v", resp["projects"])
		}
	})

	t.Run("multiple projects sorted", func(t *testing.T) {
		st := newStore(t)
		for _, p := range []string{"gamma", "alpha", "beta"} {
			if err := st.CreateProject("user1", p); err != nil {
				t.Fatalf("CreateProject(%q): %v", p, err)
			}
		}
		h := ProjectsHandler(st, noopLogger{})
		w := do(h, requestWithUser("GET", "/projects", "user1"))

		if w.Code != http.StatusOK {
			t.Errorf("got %d, want 200", w.Code)
		}
		var resp map[string][]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		want := []string{"alpha", "beta", "gamma"}
		if len(resp["projects"]) != len(want) {
			t.Fatalf("len: got %d, want %d", len(resp["projects"]), len(want))
		}
		for i, v := range want {
			if resp["projects"][i] != v {
				t.Errorf("[%d]: got %q, want %q", i, resp["projects"][i], v)
			}
		}
	})
}

func TestHandleListProjects_MissingUserID(t *testing.T) {
	h := ProjectsHandler(newStore(t), noopLogger{})
	w := do(h, httptest.NewRequest("GET", "/projects", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- DeleteProject ----

func TestHandleDeleteProject(t *testing.T) {
	tests := []struct {
		name       string
		project    string
		setup      func(*storage.Store)
		wantStatus int
	}{
		{
			name:    "existing project",
			project: "alpha",
			setup: func(st *storage.Store) {
				_ = st.CreateProject("user1", "alpha")
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found",
			project:    "nonexistent",
			wantStatus: http.StatusNotFound,
		},
		{
			// storage.DeleteProject does not call validateName; an invalid name
			// produces ErrProjectNotFound since the file cannot exist on disk.
			name:       "invalid name yields 404",
			project:    "!!bad!!",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := newStore(t)
			if tc.setup != nil {
				tc.setup(st)
			}
			h := ProjectsHandler(st, noopLogger{})
			w := do(h, requestWithUser("DELETE", "/projects/"+tc.project, "user1"))
			if w.Code != tc.wantStatus {
				t.Errorf("status: got %d, want %d (body: %s)", w.Code, tc.wantStatus, w.Body)
			}
		})
	}
}

func TestHandleDeleteProject_MissingUserID(t *testing.T) {
	h := ProjectsHandler(newStore(t), noopLogger{})
	w := do(h, httptest.NewRequest("DELETE", "/projects/alpha", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

// ---- End-to-end ----

func TestEndToEnd(t *testing.T) {
	st := newStore(t)
	h := ProjectsHandler(st, noopLogger{})

	w := do(h, requestWithUser("POST", "/projects/myproject", "user1"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201", w.Code)
	}

	w = do(h, requestWithUser("GET", "/projects", "user1"))
	var listResp map[string][]string
	_ = json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp["projects"]) != 1 || listResp["projects"][0] != "myproject" {
		t.Fatalf("list after create: got %v", listResp["projects"])
	}

	w = do(h, requestWithUser("DELETE", "/projects/myproject", "user1"))
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", w.Code)
	}

	w = do(h, requestWithUser("GET", "/projects", "user1"))
	listResp = nil
	_ = json.NewDecoder(w.Body).Decode(&listResp)
	if len(listResp["projects"]) != 0 {
		t.Fatalf("list after delete: got %v", listResp["projects"])
	}
}

// ---- Internal error (500) paths ----

// breakStore makes the storage root directory unwritable so that
// CreateProject (which calls MkdirAll) returns an unexpected error.
func breakStore(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
}

func TestHandleCreateProject_InternalError(t *testing.T) {
	tmp := t.TempDir()
	st, err := storage.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	// Make the root unwritable so MkdirAll inside CreateProject fails.
	breakStore(t, tmp)

	h := ProjectsHandler(st, noopLogger{})
	w := do(h, requestWithUser("POST", "/projects/alpha", "user1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

func TestHandleListProjects_InternalError(t *testing.T) {
	tmp := t.TempDir()
	st, err := storage.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	// Create the user dir first so it exists, then make it unreadable.
	if err := st.CreateProject("user1", "alpha"); err != nil {
		t.Fatal(err)
	}
	userDir := filepath.Join(tmp, "user1")
	if err := os.Chmod(userDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(userDir, 0o700) })

	h := ProjectsHandler(st, noopLogger{})
	w := do(h, requestWithUser("GET", "/projects", "user1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

func TestHandleDeleteProject_InternalError(t *testing.T) {
	tmp := t.TempDir()
	st, err := storage.New(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject("user1", "alpha"); err != nil {
		t.Fatal(err)
	}
	// Make the user dir unwritable so os.Remove fails.
	userDir := filepath.Join(tmp, "user1")
	if err := os.Chmod(userDir, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(userDir, 0o700) })

	h := ProjectsHandler(st, noopLogger{})
	w := do(h, requestWithUser("DELETE", "/projects/alpha", "user1"))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500", w.Code)
	}
}

func TestUserIsolation(t *testing.T) {
	st := newStore(t)
	h := ProjectsHandler(st, noopLogger{})

	w := do(h, requestWithUser("POST", "/projects/alpha", "SHA256:user1fingerprint"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d", w.Code)
	}

	w = do(h, requestWithUser("GET", "/projects", "SHA256:user2fingerprint"))
	var resp map[string][]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if len(resp["projects"]) != 0 {
		t.Fatalf("user2 should see no projects, got %v", resp["projects"])
	}
}
