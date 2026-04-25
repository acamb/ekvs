// Package client provides a typed HTTP client for the EKVS server API.
// It handles request signing, JSON encoding/decoding, and maps HTTP status
// codes to typed sentinel errors.
package client

import (
	"errors"
	"fmt"
)

// ErrUnauthorized is returned when the server responds with 401.
var ErrUnauthorized = errors.New("unauthorized")

// ErrNotFound is returned when the server responds with 404.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when the server responds with 409.
var ErrConflict = errors.New("conflict")

// ServerError is returned for any unexpected non-2xx status code.
type ServerError struct {
	StatusCode int
	Body       string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error %d: %s", e.StatusCode, e.Body)
}

// ErrServer wraps a ServerError for sentinel matching via errors.As.
var ErrServer = errors.New("server error")
