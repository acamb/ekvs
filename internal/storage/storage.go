package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

const currentVersion = 1

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]{1,128}$`)

// projectFile is the on-disk representation of a single project.
type projectFile struct {
	Version int               `json:"version"`
	Entries map[string]string `json:"entries"`
}

// Store is a file-backed storage engine for encrypted key-value pairs.
// It is safe for concurrent use within a single process.
type Store struct {
	dir   string
	mu    sync.Mutex
	locks map[string]*sync.RWMutex
}

// New creates (or opens) a Store rooted at dir.
// dir and all parent directories are created if they do not exist.
func New(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("storage.New: %w", err)
	}
	return &Store{
		dir:   dir,
		locks: make(map[string]*sync.RWMutex),
	}, nil
}

// projectLock returns (lazily creating) the per-project RWMutex.
func (s *Store) projectLock(userID, project string) *sync.RWMutex {
	key := sanitizeID(userID) + "/" + project
	s.mu.Lock()
	mu, ok := s.locks[key]
	if !ok {
		mu = &sync.RWMutex{}
		s.locks[key] = mu
	}
	s.mu.Unlock()
	return mu
}

// removeLock removes the per-project lock entry from the map.
func (s *Store) removeLock(userID, project string) {
	key := sanitizeID(userID) + "/" + project
	s.mu.Lock()
	delete(s.locks, key)
	s.mu.Unlock()
}

// validateName returns ErrInvalidName if s does not match the naming regex.
func validateName(name string) error {
	if !nameRE.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}

// sanitizeID replaces every character outside [a-zA-Z0-9_\-] with '_'.
func sanitizeID(userID string) string {
	safe := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	return safe.ReplaceAllString(userID, "_")
}

// projectPath returns the full filesystem path for a project file.
func projectPath(root, userID, project string) string {
	return filepath.Join(root, sanitizeID(userID), project+".json")
}

// readProject reads and unmarshals a project file from path.
func readProject(path string) (*projectFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("readProject: %w", err)
	}
	var pf projectFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("readProject: %w", err)
	}
	if pf.Version != currentVersion {
		return nil, fmt.Errorf("readProject: version %d: %w", pf.Version, ErrUnknownVersion)
	}
	return &pf, nil
}

// writeProject atomically writes pf to path (temp file + rename).
func writeProject(path string, pf *projectFile) error {
	data, err := json.Marshal(pf)
	if err != nil {
		return fmt.Errorf("writeProject marshal: %w", err)
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-project-*")
	if err != nil {
		return fmt.Errorf("writeProject create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Clean up temp file on any error path.
		if _, statErr := os.Stat(tmpPath); statErr == nil {
			os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("writeProject chmod: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writeProject write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writeProject close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("writeProject rename: %w", err)
	}
	return nil
}
