package storage

import (
	"fmt"
	"sort"
)

// SetSecret creates or overwrites key with value in project.
func (s *Store) SetSecret(userID, project, key, value string) error {
	if err := validateName(key); err != nil {
		return err
	}
	mu := s.projectLock(userID, project)
	mu.Lock()
	defer mu.Unlock()

	path := projectPath(s.dir, userID, project)
	pf, err := readProject(path)
	if err != nil {
		return fmt.Errorf("SetSecret: %w", err)
	}
	pf.Entries[key] = value
	return writeProject(path, pf)
}

// GetSecret returns the value stored under key in project.
func (s *Store) GetSecret(userID, project, key string) (string, error) {
	mu := s.projectLock(userID, project)
	mu.RLock()
	defer mu.RUnlock()

	path := projectPath(s.dir, userID, project)
	pf, err := readProject(path)
	if err != nil {
		return "", fmt.Errorf("GetSecret: %w", err)
	}
	val, ok := pf.Entries[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return val, nil
}

// DeleteSecret removes key from project.
func (s *Store) DeleteSecret(userID, project, key string) error {
	mu := s.projectLock(userID, project)
	mu.Lock()
	defer mu.Unlock()

	path := projectPath(s.dir, userID, project)
	pf, err := readProject(path)
	if err != nil {
		return fmt.Errorf("DeleteSecret: %w", err)
	}
	if _, ok := pf.Entries[key]; !ok {
		return ErrKeyNotFound
	}
	delete(pf.Entries, key)
	return writeProject(path, pf)
}

// ListSecrets returns all key names in project, sorted alphabetically.
func (s *Store) ListSecrets(userID, project string) ([]string, error) {
	mu := s.projectLock(userID, project)
	mu.RLock()
	defer mu.RUnlock()

	path := projectPath(s.dir, userID, project)
	pf, err := readProject(path)
	if err != nil {
		return nil, fmt.Errorf("ListSecrets: %w", err)
	}
	keys := make([]string, 0, len(pf.Entries))
	for k := range pf.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}
