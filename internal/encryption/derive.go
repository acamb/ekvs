package encryption

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	hkdfInfo   = "ekvs-encryption-key-v1"
	derivedLen = 32
	rsaMinBits = 2048
)

// DeriveKey derives a deterministic 32-byte AES-256 key from the private key
// material of signer using HKDF-SHA256.
//
// Supported types:
//   - ed25519.PrivateKey  — IKM = first 32 bytes (seed)
//   - *ecdsa.PrivateKey   — IKM = key.D.Bytes()
//   - *rsa.PrivateKey     — IKM = all prime factors concatenated (P ∥ Q ∥ …)
//     key must have a modulus ≥ 2048 bits
//
// Any other type returns ErrUnsupportedKeyType.
func DeriveKey(signer crypto.Signer) ([]byte, error) {
	ikm, err := extractIKM(signer)
	if err != nil {
		return nil, err
	}

	r := hkdf.New(sha256.New, ikm, nil, []byte(hkdfInfo))
	key := make([]byte, derivedLen)
	// hkdf.New with sha256.New never returns an error from Read.
	_, _ = io.ReadFull(r, key)
	return key, nil
}

// extractIKM returns the raw key material for each supported key type.
func extractIKM(signer crypto.Signer) ([]byte, error) {
	switch k := signer.(type) {
	case ed25519.PrivateKey:
		// The first 32 bytes of an Ed25519 private key are the seed.
		return k[:32], nil

	case *ed25519.PrivateKey:
		// golang.org/x/crypto/ssh returns a pointer; handle both forms.
		return (*k)[:32], nil

	case *ecdsa.PrivateKey:
		return k.D.Bytes(), nil

	case *rsa.PrivateKey:
		if k.N.BitLen() < rsaMinBits {
			return nil, fmt.Errorf("%w: RSA key must be at least %d bits (got %d)",
				ErrUnsupportedKeyType, rsaMinBits, k.N.BitLen())
		}
		var buf bytes.Buffer
		for _, p := range k.Primes {
			buf.Write(p.Bytes())
		}
		return buf.Bytes(), nil

	default:
		return nil, fmt.Errorf("%w: %T", ErrUnsupportedKeyType, signer)
	}
}
