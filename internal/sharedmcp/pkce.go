package sharedmcp

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

// S256Challenge returns the PKCE code_challenge for method S256.
func S256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// VerifyS256 checks that challenge is the S256 digest of verifier.
func VerifyS256(challenge, verifier string) error {
	challenge = strings.TrimSpace(challenge)
	verifier = strings.TrimSpace(verifier)
	if challenge == "" || verifier == "" {
		return errors.New("missing pkce challenge or verifier")
	}
	if challenge != S256Challenge(verifier) {
		return errors.New("pkce verifier mismatch")
	}
	return nil
}
