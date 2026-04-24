package ssh

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"

	gossh "golang.org/x/crypto/ssh"
)

// ParsePrivateKey parses an unencrypted PEM-encoded SSH private key.
// It returns a crypto.Signer suitable for signing and the corresponding
// ssh.PublicKey suitable for verification and fingerprinting.
// Only RSA, ECDSA and Ed25519 keys are supported; any other type returns
// ErrUnsupportedKeyType.
func ParsePrivateKey(pemBytes []byte) (crypto.Signer, gossh.PublicKey, error) {
	raw, err := gossh.ParseRawPrivateKey(pemBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key PEM: %w", err)
	}

	// Verify the key is one of the supported types before accepting it.
	switch raw.(type) {
	case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey, *ed25519.PrivateKey:
		// supported
	default:
		return nil, nil, ErrUnsupportedKeyType
	}

	// All types in the allowlist implement crypto.Signer.
	signer := raw.(crypto.Signer)

	sshSigner, err := gossh.NewSignerFromKey(signer)
	if err != nil {
		return nil, nil, fmt.Errorf("creating ssh signer: %w", err)
	}

	return signer, sshSigner.PublicKey(), nil
}

// ParseAuthorizedKey parses a single line in authorized_keys format and
// returns the embedded ssh.PublicKey. The trailing comment (if any) is
// discarded.
func ParseAuthorizedKey(line []byte) (gossh.PublicKey, error) {
	pub, _, _, _, err := gossh.ParseAuthorizedKey(line)
	if err != nil {
		return nil, fmt.Errorf("parsing authorized key: %w", err)
	}
	return pub, nil
}

// Fingerprint returns the SHA-256 fingerprint of pub in "SHA256:<base64>"
// format, identical to the output of `ssh-keygen -lf`.
func Fingerprint(pub gossh.PublicKey) string {
	return gossh.FingerprintSHA256(pub)
}
