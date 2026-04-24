package ssh

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// CanonicalRequest builds the message that must be signed by the client and
// verified by the server. The format is:
//
//	{METHOD}\n{PATH}\n{unix_timestamp_seconds}
//
// METHOD must be an uppercase HTTP verb (e.g. "GET", "PUT").
// PATH must include the leading slash and must not contain a query string.
// The timestamp is expressed as UTC Unix epoch seconds in base-10.
func CanonicalRequest(method, path string, ts time.Time) string {
	return strings.Join([]string{
		strings.ToUpper(method),
		path,
		strconv.FormatInt(ts.UTC().Unix(), 10),
	}, "\n")
}

// Sign signs message with signer and returns a serialised signature blob.
// The blob is produced by ssh.Marshal-ing the ssh.Signature struct, keeping
// the encoding entirely within the golang.org/x/crypto/ssh ecosystem.
func Sign(signer crypto.Signer, message []byte) ([]byte, error) {
	sshSigner, err := gossh.NewSignerFromSigner(signer)
	if err != nil {
		return nil, fmt.Errorf("creating ssh signer: %w", err)
	}

	sig, err := sshSigner.Sign(rand.Reader, message)
	if err != nil {
		return nil, fmt.Errorf("signing message: %w", err)
	}

	return gossh.Marshal(sig), nil
}
