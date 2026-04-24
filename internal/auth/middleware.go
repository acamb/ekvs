package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	internalssh "ekvs/internal/ssh"
)

// ctxKey is an unexported type for context keys in this package,
// preventing collisions with keys from other packages.
type ctxKey struct{}

var ctxKeyUserID = ctxKey{}

// AuthMiddleware returns an http.Handler that authenticates every request
// using SSH request signatures. The standard production window is 30*time.Second.
//
// Required headers:
//   - X-Timestamp:   Unix seconds (decimal string)
//   - X-Fingerprint: canonical SHA-256 fingerprint (e.g. "SHA256:...")
//   - X-Signature:   base64-encoded ssh.Marshal(sig) blob
//
// On success the verified fingerprint is injected into the request context
// and next is called. On any failure a 401 is returned with a JSON error body.
func AuthMiddleware(ks *KeyStore, window time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tsStr := r.Header.Get("X-Timestamp")
		fingerprint := r.Header.Get("X-Fingerprint")
		sigB64 := r.Header.Get("X-Signature")

		if tsStr == "" || fingerprint == "" || sigB64 == "" {
			writeError(w, http.StatusUnauthorized, "missing authentication headers")
			return
		}

		tsUnix, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid timestamp")
			return
		}
		tsTime := time.Unix(tsUnix, 0)

		pub, err := ks.Lookup(fingerprint)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unknown key")
			return
		}

		message := internalssh.CanonicalRequest(r.Method, r.URL.Path, tsTime)

		sigBlob, err := base64.StdEncoding.DecodeString(sigB64)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "malformed signature encoding")
			return
		}

		if err := internalssh.Verify(pub, []byte(message), sigBlob); err != nil {
			writeError(w, http.StatusUnauthorized, ErrInvalidSignature.Error())
			return
		}

		if err := internalssh.CheckTimestamp(tsTime, window); err != nil {
			if errors.Is(err, internalssh.ErrReplayDetected) {
				writeError(w, http.StatusUnauthorized, ErrReplayDetected.Error())
			} else {
				writeError(w, http.StatusUnauthorized, "timestamp check failed")
			}
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, fingerprint)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext retrieves the authenticated userID (fingerprint) from ctx.
// Returns ("", false) if the context was not populated by AuthMiddleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(string)
	return v, ok
}

// NewContextWithUserID returns a copy of ctx with userID stored under the same
// key used by AuthMiddleware. This is intended for use in tests that need to
// exercise handlers without going through full SSH authentication.
func NewContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// writeError writes a JSON error response with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Encoding errors are intentionally ignored: WriteHeader has already been
	// called, so the response status is committed and there is nothing useful
	// we can do if the body write fails.
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
