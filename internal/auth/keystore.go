package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	internalssh "ekvs/internal/ssh"

	gossh "golang.org/x/crypto/ssh"
)

// KeyStore is a read-only view of a directory containing .pub files.
// The system administrator manages keys by adding/removing files in that directory.
// It is safe for concurrent use.
type KeyStore struct {
	dir string
	mu  sync.RWMutex
}

// NewKeyStore opens a KeyStore rooted at keysDir.
// Returns an error if keysDir is not accessible.
func NewKeyStore(keysDir string) (*KeyStore, error) {
	if _, err := os.Stat(keysDir); err != nil {
		return nil, fmt.Errorf("auth.NewKeyStore: %w", err)
	}
	return &KeyStore{dir: keysDir}, nil
}

// Lookup scans all .pub files in keysDir and returns the public key whose
// FingerprintSHA256 matches fingerprint.
// Returns ErrKeyNotFound if no match is found.
func (ks *KeyStore) Lookup(fingerprint string) (gossh.PublicKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	entries, err := os.ReadDir(ks.dir)
	if err != nil {
		return nil, fmt.Errorf("auth.Lookup readdir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(ks.dir, e.Name()))
		if err != nil {
			continue // skip unreadable files
		}
		pub, err := internalssh.ParseAuthorizedKey(data)
		if err != nil {
			continue // skip unparseable files
		}
		if internalssh.Fingerprint(pub) == fingerprint {
			return pub, nil
		}
	}
	return nil, ErrKeyNotFound
}

// List returns the FingerprintSHA256 of every valid public key found in keysDir,
// sorted alphabetically. Unparseable or non-.pub files are silently skipped.
// Returns an empty non-nil slice if no valid keys are found.
func (ks *KeyStore) List() ([]string, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	entries, err := os.ReadDir(ks.dir)
	if err != nil {
		return nil, fmt.Errorf("auth.List readdir: %w", err)
	}
	var fingerprints []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(ks.dir, e.Name()))
		if err != nil {
			continue
		}
		pub, err := internalssh.ParseAuthorizedKey(data)
		if err != nil {
			continue
		}
		fingerprints = append(fingerprints, internalssh.Fingerprint(pub))
	}
	sort.Strings(fingerprints)
	if fingerprints == nil {
		fingerprints = []string{}
	}
	return fingerprints, nil
}
