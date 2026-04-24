package ssh

import "errors"

// Sentinel errors returned by the ssh package.
var (
	// ErrInvalidSignature is returned by Verify when the signature does not
	// match the provided public key and message.
	ErrInvalidSignature = errors.New("invalid signature")

	// ErrKeyNotFound is a reserved sentinel for use by the server layer when
	// a requested public key is not present in the authorized_keys store.
	ErrKeyNotFound = errors.New("key not found")

	// ErrReplayDetected is returned by CheckTimestamp when the request
	// timestamp falls outside the acceptable time window.
	ErrReplayDetected = errors.New("request timestamp outside acceptable window")

	// ErrUnsupportedKeyType is returned by ParsePrivateKey when the PEM block
	// decodes successfully but the key type is not supported (i.e. not RSA,
	// ECDSA, or Ed25519).
	ErrUnsupportedKeyType = errors.New("unsupported key type")
)
