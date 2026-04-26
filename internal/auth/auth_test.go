package auth

import (
	"crypto"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	internalssh "ekvs/internal/ssh"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

const (
	testdataDir = "../../internal/ssh/testdata"
	fpEd25519   = "SHA256:RxcFw9az42Zw2Sc026RYx4wfhjynuBVNrZMpIYzXbNk"
	fpEcdsa     = "SHA256:l5khlk2E/CyuOhs3RtMmgTKLmns1CBaY/Cun2a8+Jls"
	fpRsa       = "SHA256:SEntZDq/Xy8VAbcik2N+8U2s6ilxLg+7ZPhai3NfbBE"
)

// mustNewKeyStore creates a KeyStore from dir, failing the test on error.
func mustNewKeyStore(t *testing.T, dir string) *KeyStore {
	t.Helper()
	ks, err := NewKeyStore(dir)
	if err != nil {
		t.Fatalf("NewKeyStore: %v", err)
	}
	return ks
}

// mustParseSigner reads and parses a private key from path, failing on error.
func mustParseSigner(t *testing.T, path string) crypto.Signer {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	signer, _, err := internalssh.ParsePrivateKey(data)
	if err != nil {
		t.Fatalf("ParsePrivateKey(%q): %v", path, err)
	}
	return signer
}

// mustSign signs msg with signer, failing the test on error.
func mustSign(t *testing.T, signer crypto.Signer, msg []byte) []byte {
	t.Helper()
	blob, err := internalssh.Sign(signer, msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	return blob
}

// newKeysDir creates a temp directory and copies the requested .pub files into
// it with the given filenames. Returns the directory path.
func newKeysDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for dst, src := range files {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", src, err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), data, 0600); err != nil {
			t.Fatalf("write %s: %v", dst, err)
		}
	}
	return dir
}

// ---------------------------------------------------------------------------
// NewKeyStore
// ---------------------------------------------------------------------------

func TestNewKeyStore(t *testing.T) {
	t.Run("accessible dir", func(t *testing.T) {
		dir := t.TempDir()
		ks, err := NewKeyStore(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ks == nil {
			t.Fatal("expected non-nil KeyStore")
		}
	})

	t.Run("non-existent dir", func(t *testing.T) {
		_, err := NewKeyStore("/nonexistent/path/that/does/not/exist")
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}

// ---------------------------------------------------------------------------
// Lookup
// ---------------------------------------------------------------------------

func TestLookup(t *testing.T) {
	t.Run("key present", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{
			"alice.pub": testdataDir + "/ed25519.pub",
		})
		ks := mustNewKeyStore(t, dir)
		pub, err := ks.Lookup(fpEd25519)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if internalssh.Fingerprint(pub) != fpEd25519 {
			t.Errorf("fingerprint mismatch")
		}
	})

	t.Run("key not found", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{
			"alice.pub": testdataDir + "/ed25519.pub",
		})
		ks := mustNewKeyStore(t, dir)
		_, err := ks.Lookup(fpRsa)
		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("got %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("unparseable file skipped, valid key still found", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{
			"alice.pub": testdataDir + "/ed25519.pub",
		})
		// Add a bad .pub file.
		if err := os.WriteFile(filepath.Join(dir, "bad.pub"), []byte("not a valid key"), 0600); err != nil {
			t.Fatal(err)
		}
		ks := mustNewKeyStore(t, dir)
		pub, err := ks.Lookup(fpEd25519)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if internalssh.Fingerprint(pub) != fpEd25519 {
			t.Errorf("fingerprint mismatch")
		}
	})

	t.Run("non-.pub files ignored", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{})
		// Write a valid key content in a file without .pub extension.
		data, _ := os.ReadFile(testdataDir + "/ed25519.pub")
		if err := os.WriteFile(filepath.Join(dir, "alice.key"), data, 0600); err != nil {
			t.Fatal(err)
		}
		ks := mustNewKeyStore(t, dir)
		_, err := ks.Lookup(fpEd25519)
		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("got %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("unreadable .pub file skipped, valid key still found", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{
			"alice.pub": testdataDir + "/ed25519.pub",
		})
		// Add a .pub file with no read permission.
		locked := filepath.Join(dir, "locked.pub")
		if err := os.WriteFile(locked, []byte("ssh-ed25519 AAAA fake"), 0000); err != nil {
			t.Fatal(err)
		}
		ks := mustNewKeyStore(t, dir)
		pub, err := ks.Lookup(fpEd25519)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if internalssh.Fingerprint(pub) != fpEd25519 {
			t.Errorf("fingerprint mismatch")
		}
	})

	t.Run("readdir fails", func(t *testing.T) {
		dir := t.TempDir()
		ks, err := NewKeyStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		// Remove read permission on the directory.
		if err := os.Chmod(dir, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
		_, err = ks.Lookup(fpEd25519)
		if err == nil {
			t.Error("expected error when dir is not readable")
		}
	})

	t.Run("multiple key types", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{
			"alice.pub": testdataDir + "/ed25519.pub",
			"bob.pub":   testdataDir + "/ecdsa.pub",
			"carol.pub": testdataDir + "/rsa.pub",
		})
		ks := mustNewKeyStore(t, dir)
		for _, fp := range []string{fpEd25519, fpEcdsa, fpRsa} {
			if _, err := ks.Lookup(fp); err != nil {
				t.Errorf("Lookup(%q): %v", fp, err)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		ks := mustNewKeyStore(t, t.TempDir())
		fps, err := ks.List()
		if err != nil {
			t.Fatal(err)
		}
		if fps == nil {
			t.Error("expected non-nil slice")
		}
		if len(fps) != 0 {
			t.Errorf("expected empty, got %v", fps)
		}
	})

	t.Run("one valid key", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{"alice.pub": testdataDir + "/ed25519.pub"})
		ks := mustNewKeyStore(t, dir)
		fps, err := ks.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(fps) != 1 || fps[0] != fpEd25519 {
			t.Errorf("got %v, want [%s]", fps, fpEd25519)
		}
	})

	t.Run("multiple keys sorted", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{
			"alice.pub": testdataDir + "/ed25519.pub",
			"bob.pub":   testdataDir + "/ecdsa.pub",
			"carol.pub": testdataDir + "/rsa.pub",
		})
		ks := mustNewKeyStore(t, dir)
		fps, err := ks.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(fps) != 3 {
			t.Fatalf("expected 3 fingerprints, got %d", len(fps))
		}
		for i := 1; i < len(fps); i++ {
			if fps[i] < fps[i-1] {
				t.Errorf("not sorted: %v", fps)
			}
		}
	})

	t.Run("mix of valid, unparseable, and non-.pub files", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{"alice.pub": testdataDir + "/ed25519.pub"})
		if err := os.WriteFile(filepath.Join(dir, "bad.pub"), []byte("garbage"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("ssh-ed25519 AAAA..."), 0600); err != nil {
			t.Fatal(err)
		}
		ks := mustNewKeyStore(t, dir)
		fps, err := ks.List()
		if err != nil {
			t.Fatal(err)
		}
		if len(fps) != 1 || fps[0] != fpEd25519 {
			t.Errorf("got %v, want [%s]", fps, fpEd25519)
		}
	})

	t.Run("readdir fails", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("os.Chmod directory permissions not enforced on Windows")
		}
		dir := t.TempDir()
		ks := mustNewKeyStore(t, dir)
		if err := os.Chmod(dir, 0000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0700) })
		_, err := ks.List()
		if err == nil {
			t.Error("expected error when dir is not readable")
		}
	})
}

// ---------------------------------------------------------------------------
// AuthMiddleware
// ---------------------------------------------------------------------------

// makeHandler builds an AuthMiddleware-wrapped handler that records whether
// it was reached and which userID was in context.
func makeAuthTest(t *testing.T, ks *KeyStore, window time.Duration) (handler http.Handler, reached *bool, gotUserID *string) {
	t.Helper()
	var r bool
	var uid string
	reached = &r
	gotUserID = &uid
	next := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r = true
		uid, _ = UserIDFromContext(req.Context())
		w.WriteHeader(http.StatusOK)
	})
	return AuthMiddleware(ks, window, next), reached, gotUserID
}

func TestAuthMiddleware(t *testing.T) {
	dir := newKeysDir(t, map[string]string{"alice.pub": testdataDir + "/ed25519.pub"})
	ks := mustNewKeyStore(t, dir)
	signer := mustParseSigner(t, testdataDir+"/ed25519")

	const method = "GET"
	const path = "/projects/test"
	freshWindow := 30 * time.Second

	makeReq := func(tsUnix int64, fp, sigB64 string) *http.Request {
		req := httptest.NewRequest(method, path, nil)
		if tsUnix != 0 {
			req.Header.Set("X-Timestamp", fmt.Sprintf("%d", tsUnix))
		}
		if fp != "" {
			req.Header.Set("X-Fingerprint", fp)
		}
		if sigB64 != "" {
			req.Header.Set("X-Signature", sigB64)
		}
		return req
	}

	sign := func(ts time.Time) (string, int64) {
		msg := internalssh.CanonicalRequest(method, path, ts)
		blob := mustSign(t, signer, []byte(msg))
		return base64.StdEncoding.EncodeToString(blob), ts.Unix()
	}

	t.Run("AM-1 valid request", func(t *testing.T) {
		ts := time.Now().UTC()
		sigB64, tsUnix := sign(ts)
		handler, reached, uid := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq(tsUnix, fpEd25519, sigB64))
		if rr.Code != http.StatusOK {
			t.Errorf("got %d, want 200", rr.Code)
		}
		if !*reached {
			t.Error("next handler not called")
		}
		if *uid != fpEd25519 {
			t.Errorf("context userID = %q, want %q", *uid, fpEd25519)
		}
	})

	t.Run("AM-2 missing X-Timestamp", func(t *testing.T) {
		ts := time.Now().UTC()
		sigB64, tsUnix := sign(ts)
		req := makeReq(tsUnix, fpEd25519, sigB64)
		req.Header.Del("X-Timestamp")
		handler, reached, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
		if *reached {
			t.Error("next handler should not be called")
		}
	})

	t.Run("AM-3 missing X-Fingerprint", func(t *testing.T) {
		ts := time.Now().UTC()
		sigB64, tsUnix := sign(ts)
		req := makeReq(tsUnix, fpEd25519, sigB64)
		req.Header.Del("X-Fingerprint")
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("AM-4 missing X-Signature", func(t *testing.T) {
		ts := time.Now().UTC()
		_, tsUnix := sign(ts)
		req := makeReq(tsUnix, fpEd25519, "")
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("AM-5 invalid timestamp format", func(t *testing.T) {
		ts := time.Now().UTC()
		sigB64, _ := sign(ts)
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("X-Timestamp", "not-a-number")
		req.Header.Set("X-Fingerprint", fpEd25519)
		req.Header.Set("X-Signature", sigB64)
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("AM-6 unknown fingerprint", func(t *testing.T) {
		ts := time.Now().UTC()
		sigB64, tsUnix := sign(ts)
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq(tsUnix, fpRsa, sigB64))
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("AM-7 wrong signature", func(t *testing.T) {
		ts := time.Now().UTC()
		_, tsUnix := sign(ts)
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq(tsUnix, fpEd25519, base64.StdEncoding.EncodeToString([]byte("invalidsig"))))
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("AM-8 expired timestamp (>30s)", func(t *testing.T) {
		oldTs := time.Now().UTC().Add(-60 * time.Second)
		sigB64, tsUnix := sign(oldTs)
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq(tsUnix, fpEd25519, sigB64))
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})

	t.Run("malformed base64 signature", func(t *testing.T) {
		ts := time.Now().UTC()
		_, tsUnix := sign(ts)
		handler, _, _ := makeAuthTest(t, ks, freshWindow)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, makeReq(tsUnix, fpEd25519, "!!!not-base64!!!"))
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", rr.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// UserIDFromContext
// ---------------------------------------------------------------------------

func TestUserIDFromContext(t *testing.T) {
	t.Run("populated context", func(t *testing.T) {
		dir := newKeysDir(t, map[string]string{"alice.pub": testdataDir + "/ed25519.pub"})
		ks := mustNewKeyStore(t, dir)
		signer := mustParseSigner(t, testdataDir+"/ed25519")

		ts := time.Now().UTC()
		msg := internalssh.CanonicalRequest("GET", "/test", ts)
		blob := mustSign(t, signer, []byte(msg))
		sigB64 := base64.StdEncoding.EncodeToString(blob)

		var gotUID string
		var gotOK bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUID, gotOK = UserIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		handler := AuthMiddleware(ks, 30*time.Second, next)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Timestamp", fmt.Sprintf("%d", ts.Unix()))
		req.Header.Set("X-Fingerprint", fpEd25519)
		req.Header.Set("X-Signature", sigB64)
		handler.ServeHTTP(httptest.NewRecorder(), req)

		if !gotOK || gotUID != fpEd25519 {
			t.Errorf("UserIDFromContext = (%q, %v), want (%q, true)", gotUID, gotOK, fpEd25519)
		}
	})

	t.Run("empty context", func(t *testing.T) {
		uid, ok := UserIDFromContext(t.Context())
		if ok || uid != "" {
			t.Errorf("got (%q, %v), want (\"\", false)", uid, ok)
		}
	})
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestConcurrency(t *testing.T) {
	dir := newKeysDir(t, map[string]string{
		"alice.pub": testdataDir + "/ed25519.pub",
		"bob.pub":   testdataDir + "/ecdsa.pub",
		"carol.pub": testdataDir + "/rsa.pub",
	})
	ks := mustNewKeyStore(t, dir)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n * 3)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if _, err := ks.Lookup(fpEd25519); err != nil {
				t.Errorf("Lookup: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if _, err := ks.List(); err != nil {
				t.Errorf("List: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			// Best-effort mix to exercise concurrent access; errors are
			// checked exhaustively in the dedicated goroutines above.
			_, _ = ks.Lookup(fpRsa)
			_, _ = ks.List()
		}()
	}
	wg.Wait()
}
