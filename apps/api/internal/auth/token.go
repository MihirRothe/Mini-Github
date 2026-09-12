package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateRandomToken generates cryptographically secure random bytes encoded as hex.
func GenerateRandomToken(byteLen int) (string, error) {
	bytes := make([]byte, byteLen)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken computes a SHA-256 hex digest of a raw token string for secure database persistence.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// GeneratePAT creates a Personal Access Token with prefix, returning the raw token and its digest.
func GeneratePAT() (rawToken string, prefix string, digest string, err error) {
	randomHex, err := GenerateRandomToken(20) // 40 chars hex
	if err != nil {
		return "", "", "", err
	}

	rawToken = "fh_pat_" + randomHex
	prefix = rawToken[:12] // "fh_pat_xxxx" for identification in UI
	digest = HashToken(rawToken)

	return rawToken, prefix, digest, nil
}
