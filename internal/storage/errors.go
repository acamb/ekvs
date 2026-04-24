package storage

import "errors"

// Error sentinels.
var (
	ErrProjectNotFound      = errors.New("project not found")
	ErrProjectAlreadyExists = errors.New("project already exists")
	ErrKeyNotFound          = errors.New("key not found")
	ErrInvalidName          = errors.New("invalid name")
	ErrUnknownVersion       = errors.New("unknown storage version")
)
