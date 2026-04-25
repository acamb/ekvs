package ssh

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"

	gossh "golang.org/x/crypto/ssh"
)

// allowedSigner checks that raw is one of the allowed key types and returns it
// as a crypto.Signer together with the ssh.PublicKey.
func allowedSigner(raw interface{}) (crypto.Signer, gossh.PublicKey, error) {
	switch raw.(type) {
	case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey, *ed25519.PrivateKey:
		// supported
	default:
		return nil, nil, ErrUnsupportedKeyType
	}

	signer := raw.(crypto.Signer)
	sshSigner, err := gossh.NewSignerFromKey(signer)
	if err != nil {
		return nil, nil, fmt.Errorf("creating ssh signer: %w", err)
	}
	return signer, sshSigner.PublicKey(), nil
}

// ParsePrivateKey parses an unencrypted PEM-encoded SSH private key.
// It returns a crypto.Signer suitable for signing and the corresponding
// ssh.PublicKey suitable for verification and fingerprinting.
// Only RSA, ECDSA and Ed25519 keys are supported; any other type returns
// ErrUnsupportedKeyType.
// If the key is passphrase-protected, ErrPassphraseRequired is returned.
func ParsePrivateKey(pemBytes []byte) (crypto.Signer, gossh.PublicKey, error) {
	raw, err := gossh.ParseRawPrivateKey(pemBytes)
	if err != nil {
		var passErr *gossh.PassphraseMissingError
		if errors.As(err, &passErr) {
			return nil, nil, ErrPassphraseRequired
		}
		return nil, nil, fmt.Errorf("parsing private key PEM: %w", err)
	}
	return allowedSigner(raw)
}

// ParsePrivateKeyWithPassphrase parses a PEM-encoded SSH private key that may
// be encrypted with a passphrase.  It applies the same key-type allowlist as
// ParsePrivateKey.
func ParsePrivateKeyWithPassphrase(pemBytes, passphrase []byte) (crypto.Signer, gossh.PublicKey, error) {
	raw, err := gossh.ParseRawPrivateKeyWithPassphrase(pemBytes, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key with passphrase: %w", err)
	}
	return allowedSigner(raw)
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
