// Package config handles loading and saving the EKVS TUI configuration file.
package config

import (
	"errors"
	"fmt"
	"os"

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

// DefaultProfile returns a Profile populated with default values.
func DefaultProfile() Profile {
	return Profile{
		ServerURL:    "http://127.0.0.1:8080",
		IdentityFile: "~/.ssh/id_ed25519",
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
