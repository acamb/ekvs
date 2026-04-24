package auth

import (
	"errors"
	"fmt"

	internalssh "ekvs/internal/ssh"
)

// Error sentinels.
var (
	// ErrKeyNotFound is returned by Lookup when no .pub file matches the fingerprint.
	ErrKeyNotFound = errors.New("key not found")

	// ErrInvalidSignature is returned by AuthMiddleware when the signature does
	// not verify. It wraps internal/ssh.ErrInvalidSignature.
	ErrInvalidSignature = fmt.Errorf("%w", internalssh.ErrInvalidSignature)

	// ErrReplayDetected is returned by AuthMiddleware when the timestamp falls
	// outside the acceptable window. It wraps internal/ssh.ErrReplayDetected.
	ErrReplayDetected = fmt.Errorf("%w", internalssh.ErrReplayDetected)
)
