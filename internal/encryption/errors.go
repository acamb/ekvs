package encryption

import "errors"

// Sentinel errors returned by the encryption package.
var (
	// ErrUnsupportedKeyType is returned by DeriveKey when the underlying
	// crypto.Signer type is not RSA, ECDSA, or Ed25519, or when the RSA
	// key modulus is smaller than 2048 bits.
	ErrUnsupportedKeyType = errors.New("unsupported key type")

	// ErrDecryptionFailed is returned by Decrypt on any failure: invalid
	// base64, blob too short, or GCM authentication tag mismatch.
	ErrDecryptionFailed = errors.New("decryption failed")
)
