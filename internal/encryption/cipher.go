package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const nonceSize = 12 // GCM standard nonce length

// Encrypt encrypts plaintext using AES-256-GCM with a freshly generated random
// nonce. The returned string is the standard base64 encoding of:
//
//	NONCE (12 bytes) ∥ CIPHERTEXT ∥ TAG (16 bytes)
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating AES cipher: %w", err)
	}

	// cipher.NewGCM never fails for a standard AES block cipher (block size
	// is always 16 bytes); the error is checked for correctness only.
	gcm, _ := cipher.NewGCM(block)

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	// gcm.Seal appends ciphertext+tag to nonce, giving us nonce∥ciphertext∥tag.
	blob := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(blob), nil
}

// Decrypt decrypts a base64-encoded blob produced by Encrypt.
// Returns ErrDecryptionFailed on any failure (bad base64, short blob, tag
// mismatch, wrong key).
func Decrypt(key []byte, encoded string) ([]byte, error) {
	blob, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: base64 decode: %s", ErrDecryptionFailed, err)
	}

	if len(blob) < nonceSize {
		return nil, fmt.Errorf("%w: blob too short (%d bytes)", ErrDecryptionFailed, len(blob))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: creating AES cipher: %s", ErrDecryptionFailed, err)
	}

	// cipher.NewGCM never fails for a standard AES block cipher.
	gcm, _ := cipher.NewGCM(block)

	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}
