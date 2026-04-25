// Package auth provides the TUI authentication flow and request-signing helpers.
package auth

import (
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/session"
)

// ErrNotAuthenticated is returned by SignRequest when the session has no
// loaded identity.
var ErrNotAuthenticated = errors.New("not authenticated")

// SignRequest produces the three HTTP headers required by the server's
// AuthMiddleware for the given method and path at the given time.
//
// Returned map keys: "X-Timestamp", "X-Fingerprint", "X-Signature".
func SignRequest(sess *session.Session, method, path string, now time.Time) (map[string]string, error) {
	if !sess.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	canonical := internalssh.CanonicalRequest(method, path, now)
	sigBlob, err := internalssh.Sign(sess.Signer, []byte(canonical))
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"X-Timestamp":   strconv.FormatInt(now.UTC().Unix(), 10),
		"X-Fingerprint": sess.Fingerprint,
		"X-Signature":   base64.StdEncoding.EncodeToString(sigBlob),
	}, nil
}
