// Package projects implements the bubbletea model for the EKVS project
// management screen.
package projects

import "errors"

// BackMsg is emitted when the user exits the Projects screen.
// The root model handles it by returning to the main menu.
type BackMsg struct{}

// OpenSecretsMsg is emitted when the user opens a project's secrets screen.
type OpenSecretsMsg struct {
	Project string
}

// FetchedMsg carries the project list returned by the server.
type FetchedMsg struct {
	Projects []string
}

// ErrMsg carries an error that occurred during an API operation.
type ErrMsg struct {
	Err error
}

func (e ErrMsg) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

// Unwrap allows errors.Is / errors.As to inspect the wrapped error.
func (e ErrMsg) Unwrap() error { return e.Err }

// ensure ErrMsg satisfies the error interface
var _ error = ErrMsg{Err: errors.New("")}
