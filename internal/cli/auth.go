package cli

import (
	"crypto"
	"errors"
	"fmt"
	"os"

	internalssh "ekvs/internal/ssh"

	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// LoadIdentity loads an SSH private key from identityPath.
// passphrase is tried first; if empty and the key is encrypted, falls back to prompting
// on stderr with echo disabled.
// The passphrase argument resolution is handled in the root command.
// Returns (signer, pubKey, fingerprint, error).
func LoadIdentity(identityPath, passphrase string) (crypto.Signer, gossh.PublicKey, string, error) {
	pemBytes, err := os.ReadFile(identityPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("read identity %q: %w", identityPath, err)
	}

	signer, pub, err := internalssh.ParsePrivateKey(pemBytes)
	if err == nil {
		return signer, pub, internalssh.Fingerprint(pub), nil
	}

	if !errors.Is(err, internalssh.ErrPassphraseRequired) {
		return nil, nil, "", fmt.Errorf("parse identity %q: %w", identityPath, err)
	}

	// Key is encrypted — resolve passphrase.
	if passphrase == "" {
		fmt.Fprintf(os.Stderr, "Enter passphrase for %s: ", identityPath)
		raw, readErr := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if readErr != nil {
			return nil, nil, "", fmt.Errorf("read passphrase: %w", readErr)
		}
		passphrase = string(raw)
	}

	signer, pub, err = internalssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(passphrase))
	if err != nil {
		return nil, nil, "", fmt.Errorf("parse identity %q: %w", identityPath, err)
	}
	return signer, pub, internalssh.Fingerprint(pub), nil
}
