package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

const testUser = "SHA256:testfingerprint"

// helpers

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func mustCreateProject(t *testing.T, s *Store, userID, project string) {
	t.Helper()
	if err := s.CreateProject(userID, project); err != nil {
		t.Fatalf("CreateProject(%q, %q): %v", userID, project, err)
	}
}

// --- New ---

func TestNew(t *testing.T) {
	t.Run("fresh directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "subdir", "storage")
		s, err := New(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s == nil {
			t.Fatal("expected non-nil Store")
		}
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("directory not created: %v", err)
		}
	})

	t.Run("pre-existing directory", func(t *testing.T) {
		dir := t.TempDir()
		// Write a sentinel file to ensure it is untouched.
		sentinel := filepath.Join(dir, "sentinel.txt")
		if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := New(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if data, err := os.ReadFile(sentinel); err != nil || string(data) != "keep" {
			t.Fatal("existing file was modified")
		}
	})
}

// --- CreateProject ---

func TestCreateProject(t *testing.T) {
	tests := []struct {
		name    string
		project string
		wantErr error
	}{
		{"valid", "myproject", nil},
		{"invalid empty", "", ErrInvalidName},
		{"invalid slash", "a/b", ErrInvalidName},
		{"invalid too long", fmt.Sprintf("%0129d", 0), ErrInvalidName},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t)
			err := s.CreateProject(testUser, tc.project)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}

	t.Run("duplicate", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		if err := s.CreateProject(testUser, "proj"); !errors.Is(err, ErrProjectAlreadyExists) {
			t.Errorf("got %v, want ErrProjectAlreadyExists", err)
		}
	})

	t.Run("file mode 0600", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		path := projectPath(s.dir, testUser, "proj")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("file mode %o, want 0600", perm)
		}
	})
}

// --- DeleteProject ---

func TestDeleteProject(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		if err := s.DeleteProject(testUser, "proj"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		path := projectPath(s.dir, testUser, "proj")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Error("file should be gone")
		}
	})

	t.Run("not found", func(t *testing.T) {
		s := newStore(t)
		if err := s.DeleteProject(testUser, "ghost"); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("got %v, want ErrProjectNotFound", err)
		}
	})

	t.Run("lock removed from map", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		// Force lock creation.
		_ = s.projectLock(testUser, "proj")
		key := sanitizeID(testUser) + "/proj"
		s.mu.Lock()
		if _, ok := s.locks[key]; !ok {
			s.mu.Unlock()
			t.Fatal("lock should exist before delete")
		}
		s.mu.Unlock()

		if err := s.DeleteProject(testUser, "proj"); err != nil {
			t.Fatal(err)
		}
		s.mu.Lock()
		_, stillPresent := s.locks[key]
		s.mu.Unlock()
		if stillPresent {
			t.Error("lock entry should be removed after DeleteProject")
		}
	})
}

// --- ListProjects ---

func TestListProjects(t *testing.T) {
	t.Run("user dir does not exist", func(t *testing.T) {
		s := newStore(t)
		names, err := s.ListProjects("SHA256:nobody")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if names == nil {
			t.Error("expected non-nil slice")
		}
		if len(names) != 0 {
			t.Errorf("expected empty, got %v", names)
		}
	})

	t.Run("user with no projects", func(t *testing.T) {
		s := newStore(t)
		// Create user dir without any files.
		userDir := filepath.Join(s.dir, sanitizeID(testUser))
		if err := os.MkdirAll(userDir, 0700); err != nil {
			t.Fatal(err)
		}
		names, err := s.ListProjects(testUser)
		if err != nil {
			t.Fatal(err)
		}
		if len(names) != 0 {
			t.Errorf("expected empty, got %v", names)
		}
	})

	t.Run("single project", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "alpha")
		names, err := s.ListProjects(testUser)
		if err != nil {
			t.Fatal(err)
		}
		if len(names) != 1 || names[0] != "alpha" {
			t.Errorf("got %v, want [alpha]", names)
		}
	})

	t.Run("multiple projects sorted", func(t *testing.T) {
		s := newStore(t)
		for _, p := range []string{"charlie", "alpha", "bravo"} {
			mustCreateProject(t, s, testUser, p)
		}
		names, err := s.ListProjects(testUser)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"alpha", "bravo", "charlie"}
		for i, n := range names {
			if n != want[i] {
				t.Errorf("index %d: got %q, want %q", i, n, want[i])
			}
		}
	})
}

// --- SetSecret ---

func TestSetSecret(t *testing.T) {
	t.Run("new key", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		if err := s.SetSecret(testUser, "proj", "KEY", "val"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("overwrite key", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		_ = s.SetSecret(testUser, "proj", "KEY", "old")
		if err := s.SetSecret(testUser, "proj", "KEY", "new"); err != nil {
			t.Fatal(err)
		}
		val, _ := s.GetSecret(testUser, "proj", "KEY")
		if val != "new" {
			t.Errorf("got %q, want \"new\"", val)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetSecret(testUser, "ghost", "KEY", "v"); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("got %v, want ErrProjectNotFound", err)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		if err := s.SetSecret(testUser, "proj", "bad/key", "v"); !errors.Is(err, ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})
}

// --- GetSecret ---

func TestGetSecret(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		_ = s.SetSecret(testUser, "proj", "KEY", "secret")
		val, err := s.GetSecret(testUser, "proj", "KEY")
		if err != nil {
			t.Fatal(err)
		}
		if val != "secret" {
			t.Errorf("got %q, want \"secret\"", val)
		}
	})

	t.Run("key not found", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		if _, err := s.GetSecret(testUser, "proj", "MISSING"); !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("got %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		s := newStore(t)
		if _, err := s.GetSecret(testUser, "ghost", "KEY"); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("got %v, want ErrProjectNotFound", err)
		}
	})
}

// --- DeleteSecret ---

func TestDeleteSecret(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		_ = s.SetSecret(testUser, "proj", "KEY", "val")
		if err := s.DeleteSecret(testUser, "proj", "KEY"); err != nil {
			t.Fatal(err)
		}
		if _, err := s.GetSecret(testUser, "proj", "KEY"); !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("key should be deleted, got %v", err)
		}
	})

	t.Run("key not found", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		if err := s.DeleteSecret(testUser, "proj", "MISSING"); !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("got %v, want ErrKeyNotFound", err)
		}
	})

	t.Run("project not found", func(t *testing.T) {
		s := newStore(t)
		if err := s.DeleteSecret(testUser, "ghost", "KEY"); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("got %v, want ErrProjectNotFound", err)
		}
	})
}

// --- ListSecrets ---

func TestListSecrets(t *testing.T) {
	t.Run("empty project", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		keys, err := s.ListSecrets(testUser, "proj")
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Errorf("expected empty, got %v", keys)
		}
	})

	t.Run("multiple keys sorted", func(t *testing.T) {
		s := newStore(t)
		mustCreateProject(t, s, testUser, "proj")
		for _, k := range []string{"ZEBRA", "ALPHA", "MIDDLE"} {
			_ = s.SetSecret(testUser, "proj", k, "v")
		}
		keys, err := s.ListSecrets(testUser, "proj")
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"ALPHA", "MIDDLE", "ZEBRA"}
		for i, k := range keys {
			if k != want[i] {
				t.Errorf("index %d: got %q, want %q", i, k, want[i])
			}
		}
	})

	t.Run("project not found", func(t *testing.T) {
		s := newStore(t)
		if _, err := s.ListSecrets(testUser, "ghost"); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("got %v, want ErrProjectNotFound", err)
		}
	})
}

// --- Version handling ---

func TestUnknownVersion(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")
	path := projectPath(s.dir, testUser, "proj")

	// Overwrite with unknown version.
	bad := map[string]any{"version": 99, "entries": map[string]string{}}
	data, _ := json.Marshal(bad)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, err := s.GetSecret(testUser, "proj", "any")
	if !errors.Is(err, ErrUnknownVersion) {
		t.Errorf("got %v, want ErrUnknownVersion", err)
	}
}

// --- sanitizeID ---

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SHA256:abc+def/ghi=", "SHA256_abc_def_ghi_"},
		{"simple", "simple"},
		{"a-b_c", "a-b_c"},
	}
	for _, tc := range tests {
		if got := sanitizeID(tc.input); got != tc.want {
			t.Errorf("sanitizeID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- New error path ---

func TestNewError(t *testing.T) {
	// Create a file where the dir should be, so MkdirAll fails.
	parent := t.TempDir()
	blocker := filepath.Join(parent, "blocked")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := New(filepath.Join(blocker, "sub"))
	if err == nil {
		t.Error("expected error when dir cannot be created")
	}
}

// --- readProject error paths ---

func TestReadProjectInvalidJSON(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")
	path := projectPath(s.dir, testUser, "proj")

	// Write invalid JSON.
	if err := os.WriteFile(path, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := s.GetSecret(testUser, "proj", "KEY")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// --- writeProject error paths ---

func TestWriteProject_CreateTempFails(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")

	// Make the user directory non-writable so os.CreateTemp fails.
	userDir := filepath.Join(s.dir, sanitizeID(testUser))
	if err := os.Chmod(userDir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(userDir, 0o700) })

	err := s.SetSecret(testUser, "proj", "key", "val")
	if err == nil {
		t.Fatal("expected error when directory is not writable, got nil")
	}
}

func TestWriteProject_RenameFails(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")

	// Make the user directory execute-only: CreateTemp succeeds (the dir
	// entry for the new file is created by the kernel) but os.Rename fails
	// because renaming requires write permission on the directory.
	userDir := filepath.Join(s.dir, sanitizeID(testUser))
	if err := os.Chmod(userDir, 0o100); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(userDir, 0o700) })

	err := s.SetSecret(testUser, "proj", "key", "val")
	if err == nil {
		t.Fatal("expected error when rename cannot proceed, got nil")
	}
}

// --- DeleteProject non-IsNotExist stat error ---

func TestDeleteProjectFileReadOnly(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")

	// Make the user dir not readable so os.Remove fails (not a "not found" error).
	userDir := filepath.Join(s.dir, sanitizeID(testUser))
	if err := os.Chmod(userDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(userDir, 0700) })

	err := s.DeleteProject(testUser, "proj")
	// We expect some error (either stat fails or remove fails).
	if err == nil {
		t.Error("expected error when directory is not writable")
	}
}

// --- ListProjects non-ErrNotExist error ---

func TestListProjectsReadError(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")

	// Remove read permission from user dir so ReadDir fails.
	userDir := filepath.Join(s.dir, sanitizeID(testUser))
	if err := os.Chmod(userDir, 0300); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(userDir, 0700) })

	_, err := s.ListProjects(testUser)
	if err == nil {
		t.Error("expected error when directory is not readable")
	}
}

// --- Persistence ---

func TestPersistence(t *testing.T) {
	dir := t.TempDir()

	// Store A: create project and set secret.
	a, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.CreateProject(testUser, "proj"); err != nil {
		t.Fatal(err)
	}
	if err := a.SetSecret(testUser, "proj", "TOKEN", "abc123"); err != nil {
		t.Fatal(err)
	}

	// Store B: independent instance on same dir.
	b, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	val, err := b.GetSecret(testUser, "proj", "TOKEN")
	if err != nil {
		t.Fatalf("GetSecret via Store B: %v", err)
	}
	if val != "abc123" {
		t.Errorf("got %q, want \"abc123\"", val)
	}

	// Store A deletes; Store B should see it gone.
	if err := a.DeleteProject(testUser, "proj"); err != nil {
		t.Fatal(err)
	}
	projects, err := b.ListProjects(testUser)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projects {
		if p == "proj" {
			t.Error("deleted project still visible via Store B")
		}
	}
}

// --- Concurrency ---

func TestConcurrencySameProject(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "shared")

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("KEY_%d", i)
			if err := s.SetSecret(testUser, "shared", key, "v"); err != nil {
				t.Errorf("SetSecret: %v", err)
			}
			if _, err := s.GetSecret(testUser, "shared", key); err != nil {
				t.Errorf("GetSecret: %v", err)
			}
		}()
	}
	wg.Wait()

	keys, err := s.ListSecrets(testUser, "shared")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != n {
		t.Errorf("expected %d keys, got %d", n, len(keys))
	}
}

func TestConcurrencyDifferentProjects(t *testing.T) {
	s := newStore(t)
	const n = 20
	for i := 0; i < n; i++ {
		mustCreateProject(t, s, testUser, fmt.Sprintf("proj_%d", i))
	}
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			proj := fmt.Sprintf("proj_%d", i)
			_ = s.SetSecret(testUser, proj, "K", "v")
			_, _ = s.GetSecret(testUser, proj, "K")
		}()
	}
	wg.Wait()
}

func TestConcurrencyReadWhileWrite(t *testing.T) {
	s := newStore(t)
	mustCreateProject(t, s, testUser, "proj")
	_ = s.SetSecret(testUser, "proj", "KEY", "initial")

	const readers = 20
	var wg sync.WaitGroup
	wg.Add(readers + 1)

	// One writer.
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = s.SetSecret(testUser, "proj", "KEY", fmt.Sprintf("val_%d", i))
		}
	}()

	// Many concurrent readers.
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				val, err := s.GetSecret(testUser, "proj", "KEY")
				if err != nil {
					t.Errorf("GetSecret: %v", err)
				}
				// Value must not be empty (either old or new, both valid).
				if val == "" {
					t.Error("read empty value during concurrent write")
				}
			}
		}()
	}
	wg.Wait()
}

func TestConcurrencyDeleteWhileAccess(t *testing.T) {
	const n = 20
	s := newStore(t)
	for i := 0; i < n; i++ {
		mustCreateProject(t, s, testUser, fmt.Sprintf("proj_%d", i))
	}

	var wg sync.WaitGroup
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		i := i
		proj := fmt.Sprintf("proj_%d", i)
		// Deleter goroutine.
		go func() {
			defer wg.Done()
			_ = s.DeleteProject(testUser, proj)
		}()
		// Accessor goroutine — may get ErrProjectNotFound, that's fine.
		go func() {
			defer wg.Done()
			err := s.SetSecret(testUser, proj, "K", "v")
			if err != nil && !errors.Is(err, ErrProjectNotFound) {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}
