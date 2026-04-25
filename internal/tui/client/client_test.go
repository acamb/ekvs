package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/session"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// testSession builds an authenticated session using the ed25519 test key
// from internal/ssh/testdata.
func testSession(t *testing.T) *session.Session {
	t.Helper()
	pemBytes, err := os.ReadFile("../../../internal/ssh/testdata/ed25519")
	if err != nil {
		t.Fatalf("read test key: %v", err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("parse test key: %v", err)
	}
	fp := internalssh.Fingerprint(pub)
	return &session.Session{Signer: signer, PublicKey: pub, Fingerprint: fp}
}

// jsonHandler returns a handler that responds with the given status and JSON body.
func jsonHandler(status int, body interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}
}

// ── ListProjects ─────────────────────────────────────────────────────────────

func TestListProjects_Success(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, map[string][]string{
		"projects": {"alpha", "beta"},
	}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	got, err := c.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Errorf("got %v, want [alpha beta]", got)
	}
}

func TestListProjects_Empty(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, map[string][]string{
		"projects": {},
	}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	got, err := c.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestListProjects_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusUnauthorized, nil))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	_, err := c.ListProjects()
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("want ErrUnauthorized, got %v", err)
	}
}

// ── CreateProject ─────────────────────────────────────────────────────────────

func TestCreateProject_Success(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusCreated, map[string]string{"name": "foo"}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	if err := c.CreateProject("foo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateProject_Conflict(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusConflict, nil))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	err := c.CreateProject("foo")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("want ErrConflict, got %v", err)
	}
}

func TestCreateProject_BadRequest(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusBadRequest, nil))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	err := c.CreateProject("bad name")
	var se *ServerError
	if !errors.As(err, &se) {
		t.Errorf("want *ServerError, got %T: %v", err, err)
	}
	if se.StatusCode != http.StatusBadRequest {
		t.Errorf("want status 400, got %d", se.StatusCode)
	}
}

// ── DeleteProject ─────────────────────────────────────────────────────────────

func TestDeleteProject_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	if err := c.DeleteProject("foo"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteProject_NotFound(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusNotFound, nil))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	err := c.DeleteProject("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// ── Auth headers ──────────────────────────────────────────────────────────────

func TestClient_SignsRequests(t *testing.T) {
	checked := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, h := range []string{"X-Timestamp", "X-Fingerprint", "X-Signature"} {
			if r.Header.Get(h) == "" {
				t.Errorf("missing header %s", h)
			}
		}
		checked = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	_ = c.DeleteProject("any")
	if !checked {
		t.Fatal("handler was never called")
	}
}

// ── Network errors ────────────────────────────────────────────────────────────

func TestClient_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // close immediately

	c := New(srv.URL, testSession(t))
	_, err := c.ListProjects()
	if err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}

func TestCreateProject_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := New(srv.URL, testSession(t))
	if err := c.CreateProject("foo"); err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}

func TestDeleteProject_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := New(srv.URL, testSession(t))
	if err := c.DeleteProject("foo"); err == nil {
		t.Fatal("expected error for closed server, got nil")
	}
}

// ── ServerError.Error() ───────────────────────────────────────────────────────

func TestServerError_Error(t *testing.T) {
	e := &ServerError{StatusCode: 500, Body: "boom"}
	got := e.Error()
	want := "server error 500: boom"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── Sign error (unauthenticated session) ──────────────────────────────────────

func TestClient_SignError_Unauthenticated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	empty := &session.Session{} // not authenticated
	c := New(srv.URL, empty)
	_, err := c.ListProjects()
	if err == nil {
		t.Fatal("expected sign error, got nil")
	}
}

// ── Decode error ──────────────────────────────────────────────────────────────

func TestListProjects_DecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	_, err := c.ListProjects()
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

// ── Null projects payload ─────────────────────────────────────────────────────

func TestListProjects_NullProjects(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(http.StatusOK, map[string]interface{}{
		"projects": nil,
	}))
	defer srv.Close()

	c := New(srv.URL, testSession(t))
	got, err := c.ListProjects()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("expected empty non-nil slice, got %v", got)
	}
}
