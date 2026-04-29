package cli

import (
	"crypto"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	internalssh "ekvs/internal/ssh"
)

// SignedHeaders produces the three HTTP authentication headers required by
// the EKVS server for the given method and path at time now.
// Returned keys: "X-Timestamp", "X-Fingerprint", "X-Signature".
func SignedHeaders(signer crypto.Signer, fingerprint, method, path string, now time.Time) (map[string]string, error) {
	canonical := internalssh.CanonicalRequest(method, path, now)
	sigBlob, err := internalssh.Sign(signer, []byte(canonical))
	if err != nil {
		return nil, fmt.Errorf("sign request: %w", err)
	}
	return map[string]string{
		"X-Timestamp":   strconv.FormatInt(now.UTC().Unix(), 10),
		"X-Fingerprint": fingerprint,
		"X-Signature":   base64.StdEncoding.EncodeToString(sigBlob),
	}, nil
}
