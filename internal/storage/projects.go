package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CreateProject creates a new empty project file for userID.
func (s *Store) CreateProject(userID, project string) error {
	if err := validateName(project); err != nil {
		return err
	}
	mu := s.projectLock(userID, project)
	mu.Lock()
	defer mu.Unlock()

	path := projectPath(s.dir, userID, project)
	if _, err := os.Stat(path); err == nil {
		return ErrProjectAlreadyExists
	}
	// Ensure user directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("CreateProject mkdir: %w", err)
	}
	pf := &projectFile{
		Version: currentVersion,
		Entries: make(map[string]string),
	}
	return writeProject(path, pf)
}

// DeleteProject removes the project file and all its secrets.
func (s *Store) DeleteProject(userID, project string) error {
	mu := s.projectLock(userID, project)
	mu.Lock()

	path := projectPath(s.dir, userID, project)
	if _, err := os.Stat(path); err != nil {
		mu.Unlock()
		if os.IsNotExist(err) {
			return ErrProjectNotFound
		}
		return fmt.Errorf("DeleteProject stat: %w", err)
	}
	err := os.Remove(path)
	mu.Unlock()
	if err != nil {
		return fmt.Errorf("DeleteProject remove: %w", err)
	}
	// Remove the lock entry from the map to prevent unbounded growth.
	s.removeLock(userID, project)
	return nil
}

// ListProjects returns all project names for userID, sorted alphabetically.
// Returns an empty non-nil slice if the user has no projects or if the
// user directory does not exist yet.
func (s *Store) ListProjects(userID string) ([]string, error) {
	userDir := filepath.Join(s.dir, sanitizeID(userID))
	entries, err := os.ReadDir(userDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("ListProjects readdir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(names)
	if names == nil {
		names = []string{}
	}
	return names, nil
}
