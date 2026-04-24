package ssh

import (
	"fmt"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// Verify verifies that sigBlob is a valid signature over message produced by
// the private key corresponding to pub. It returns ErrInvalidSignature if
// the signature is not valid.
func Verify(pub gossh.PublicKey, message, sigBlob []byte) error {
	var sig gossh.Signature
	if err := gossh.Unmarshal(sigBlob, &sig); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSignature, err.Error())
	}

	if err := pub.Verify(message, &sig); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidSignature, err.Error())
	}

	return nil
}

// CheckTimestamp returns ErrReplayDetected if ts is not strictly within
// ±window of the current UTC time (i.e. |now - ts| > window is rejected,
// and so is |now - ts| == window).
func CheckTimestamp(ts time.Time, window time.Duration) error {
	delta := time.Since(ts.UTC())
	if delta < 0 {
		delta = -delta
	}
	if delta >= window {
		return fmt.Errorf("%w: delta=%s window=%s", ErrReplayDetected, delta, window)
	}
	return nil
}
