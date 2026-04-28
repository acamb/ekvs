// Package config handles loading and saving the EKVS TUI configuration file.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile represents a single connection profile.
type Profile struct {
	Name         string `yaml:"name"`
	ServerURL    string `yaml:"server_url"`
	IdentityFile string `yaml:"identity_file"`
	Theme        string `yaml:"theme"`
}

// ConfigFile is the root structure of the YAML configuration file.
type ConfigFile struct {
	Profiles []Profile `yaml:"profiles"`
}

// ExpandHome replaces a leading "~" with the user home directory obtained via
// os.UserHomeDir(). The path is returned unchanged when it does not start with
// "~" or when the home directory cannot be determined.
// filepath.Join handles the platform separator so the result is always correct
// on both Linux/macOS and Windows.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	// Strip leading "~" (and any immediately following separator) before joining.
	return filepath.Join(home, path[1:])
}

// SSHDir returns the platform-appropriate SSH configuration directory:
// <home>/.ssh on Linux/macOS and <home>\.ssh on Windows.
// Returns an empty string and an error when os.UserHomeDir() fails.
func SSHDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("ssh dir: %w", err)
	}
	return filepath.Join(home, ".ssh"), nil
}

// DiscoverSSHKeys returns the absolute paths of likely SSH private-key files
// found in the directory returned by sshDirFn.
// If sshDirFn is nil, SSHDir is used.
// Public-key files (.pub) and well-known non-key filenames are excluded.
// Returns nil when the directory cannot be read.
func DiscoverSSHKeys(sshDirFn func() (string, error)) []string {
	fn := sshDirFn
	if fn == nil {
		fn = SSHDir
	}
	dir, err := fn()
	if err != nil || dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	skip := map[string]bool{
		"known_hosts":     true,
		"known_hosts.old": true,
		"config":          true,
		"authorized_keys": true,
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".pub") || skip[name] {
			continue
		}
		keys = append(keys, filepath.Join(dir, name))
	}
	return keys
}

// DefaultProfile returns a Profile populated with default values.
// IdentityFile is returned as an expanded absolute path so callers never need
// to handle "~" expansion themselves.
func DefaultProfile() Profile {
	return Profile{
		ServerURL:    "http://127.0.0.1:8080",
		IdentityFile: ExpandHome("~/.ssh/id_ed25519"),
		Theme:        "adaptive",
	}
}

// applyDefaults fills empty fields of p with values from DefaultProfile.
func applyDefaults(p *Profile) {
	d := DefaultProfile()
	if p.ServerURL == "" {
		p.ServerURL = d.ServerURL
	}
	if p.IdentityFile == "" {
		p.IdentityFile = d.IdentityFile
	}
	if p.Theme == "" {
		p.Theme = d.Theme
	}
}

// LoadFromFile loads a ConfigFile from the YAML file at path.
//
// Returns (nil, nil) when:
//   - required is false and the file does not exist, or
//   - the profiles list is empty.
//
// Returns an error when:
//   - required is true and the file does not exist,
//   - the file contains invalid YAML,
//   - a profile has an empty name, or
//   - two profiles share the same name.
func LoadFromFile(path string, required bool) (*ConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if required {
				return nil, fmt.Errorf("configuration file not found: %s", path)
			}
			return nil, nil
		}
		return nil, fmt.Errorf("error reading configuration file: %w", err)
	}

	var cf ConfigFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("YAML parse error in %s: %w", path, err)
	}

	if len(cf.Profiles) == 0 {
		return nil, nil
	}

	seen := map[string]bool{}
	for i := range cf.Profiles {
		p := &cf.Profiles[i]
		if p.Name == "" {
			return nil, fmt.Errorf("profile at position %d has no name", i+1)
		}
		if seen[p.Name] {
			return nil, fmt.Errorf("duplicate profile name: %q", p.Name)
		}
		seen[p.Name] = true
		applyDefaults(p)
	}

	return &cf, nil
}

// Save serialises cf to YAML and writes it to the file at path.
func Save(path string, cf *ConfigFile) error {
	data, err := yaml.Marshal(cf)
	if err != nil {
		return fmt.Errorf("YAML serialisation error: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("error writing file %s: %w", path, err)
	}
	return nil
}

// ── CRUD helpers ─────────────────────────────────────────────────────────────

// FindProfile returns the profile whose Name matches name together with its
// zero-based index in the Profiles slice. The third return value is false when
// no profile with that name exists.
func (cf *ConfigFile) FindProfile(name string) (Profile, int, bool) {
	for i, p := range cf.Profiles {
		if p.Name == name {
			return p, i, true
		}
	}
	return Profile{}, -1, false
}

// UpsertProfile performs a true upsert on the Profiles list:
//   - if no profile with the same Name exists, the profile is appended;
//   - if a profile with the same Name already exists, it is replaced in-place.
//
// applyDefaults is called on the profile before it is stored so that blank
// optional fields receive their canonical values.
// Returns an error when profile.Name is empty.
func (cf *ConfigFile) UpsertProfile(profile Profile) error {
	if profile.Name == "" {
		return errors.New("profile name cannot be empty")
	}
	applyDefaults(&profile)
	_, i, found := cf.FindProfile(profile.Name)
	if found {
		cf.Profiles[i] = profile
	} else {
		cf.Profiles = append(cf.Profiles, profile)
	}
	return nil
}

// UpdateProfile is a rename-safe update: it locates the entry by oldName,
// validates that the new name is unique (unless the name is unchanged), applies
// defaults, and replaces the entry in-place.
//
// Returns an error when:
//   - profile.Name is empty;
//   - oldName is not found in the list;
//   - the new name conflicts with a different existing profile.
func (cf *ConfigFile) UpdateProfile(oldName string, profile Profile) error {
	if profile.Name == "" {
		return errors.New("profile name cannot be empty")
	}
	_, idx, found := cf.FindProfile(oldName)
	if !found {
		return fmt.Errorf("profile %q not found", oldName)
	}
	if profile.Name != oldName {
		if _, _, exists := cf.FindProfile(profile.Name); exists {
			return fmt.Errorf("profile name %q already exists", profile.Name)
		}
	}
	applyDefaults(&profile)
	cf.Profiles[idx] = profile
	return nil
}

// DeleteProfile removes the profile with the given name from the list.
// Returns an error when no profile with that name exists.
func (cf *ConfigFile) DeleteProfile(name string) error {
	_, idx, found := cf.FindProfile(name)
	if !found {
		return fmt.Errorf("profile %q not found", name)
	}
	cf.Profiles = append(cf.Profiles[:idx], cf.Profiles[idx+1:]...)
	return nil
}
