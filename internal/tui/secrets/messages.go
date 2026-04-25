// Package secrets implements the bubbletea model for the EKVS secret
// management screen.
package secrets

import (
	"errors"

	"ekvs/internal/tui/client"
)

// BackMsg is emitted when the user exits the Secrets screen.
// The root model handles it by returning to the Projects screen.
type BackMsg struct{}

// FetchedMsg carries the secret list returned by the server.
type FetchedMsg struct {
	Secrets []client.SecretEntry
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
