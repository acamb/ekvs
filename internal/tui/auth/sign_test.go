package auth

import (
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	internalssh "ekvs/internal/ssh"
	"ekvs/internal/tui/session"
)

func loadSession(t *testing.T, keyPath string) *session.Session {
	t.Helper()
	pemBytes, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("reading key: %v", err)
	}
	signer, pub, err := internalssh.ParsePrivateKey(pemBytes)
	if err != nil {
		t.Fatalf("parsing key: %v", err)
	}
	return &session.Session{
		Signer:      signer,
		PublicKey:   pub,
		Fingerprint: internalssh.Fingerprint(pub),
	}
}

func TestSignRequest_Unauthenticated(t *testing.T) {
	s := &session.Session{}
	_, err := SignRequest(s, "GET", "/v1/projects", time.Now())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got %v", err)
	}
}

func TestSignRequest_Valid(t *testing.T) {
	s := loadSession(t, "../../../internal/ssh/testdata/ed25519")
	now := time.Now().UTC().Truncate(time.Second)

	headers, err := SignRequest(s, "GET", "/v1/projects", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify timestamp header
	tsStr, ok := headers["X-Timestamp"]
	if !ok {
		t.Fatal("missing X-Timestamp header")
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		t.Fatalf("invalid timestamp: %v", err)
	}
	if ts != now.Unix() {
		t.Errorf("timestamp mismatch: got %d want %d", ts, now.Unix())
	}

	// Verify fingerprint header
	if headers["X-Fingerprint"] != s.Fingerprint {
		t.Errorf("fingerprint mismatch")
	}

	// Decode and verify signature
	sigBlob, err := base64.StdEncoding.DecodeString(headers["X-Signature"])
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	canonical := internalssh.CanonicalRequest("GET", "/v1/projects", now)
	if err := internalssh.Verify(s.PublicKey, []byte(canonical), sigBlob); err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}
