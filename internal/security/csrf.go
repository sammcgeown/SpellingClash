package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// CSRFGenerator generates and validates CSRF tokens using HMAC-SHA256.
// Tokens are derived deterministically from the session ID and a secret key,
// so no shared state is required — safe for multi-replica / Kubernetes deployments.
type CSRFGenerator struct {
	secret []byte
}

// NewCSRFGenerator creates a new stateless HMAC-based CSRF generator.
func NewCSRFGenerator(secret string) *CSRFGenerator {
	return &CSRFGenerator{secret: []byte(secret)}
}

// GenerateToken returns a deterministic CSRF token for the given session ID.
func (g *CSRFGenerator) GenerateToken(sessionID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session ID is required")
	}
	mac := hmac.New(sha256.New, g.secret)
	mac.Write([]byte(sessionID))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// ValidateToken reports whether token is the valid CSRF token for sessionID.
func (g *CSRFGenerator) ValidateToken(sessionID, token string) bool {
	if sessionID == "" || token == "" {
		return false
	}
	expected, err := g.GenerateToken(sessionID)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(expected), []byte(token))
}
