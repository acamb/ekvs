package errors

import "errors"

// Sentinel errors.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInternal     = errors.New("internal error")
)

// AppError is a structured application error that wraps an underlying error
// and carries a machine-readable Code and a human-readable Message.
type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates an AppError with the given code, message, and optional wrapped error.
func New(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
