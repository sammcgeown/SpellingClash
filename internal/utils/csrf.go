package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// CSRFToken represents a CSRF token with expiration
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// CSRFTokenStore manages CSRF tokens in memory
// In production, consider using Redis for distributed systems
type CSRFTokenStore struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
	ttl    time.Duration
}

// NewCSRFTokenStore creates a new CSRF token store
func NewCSRFTokenStore(ttl time.Duration) *CSRFTokenStore {
	store := &CSRFTokenStore{
		tokens: make(map[string]*CSRFToken),
		ttl:    ttl,
	}
	// Start cleanup goroutine
	go store.cleanupExpired()
	return store
}

// GenerateToken generates a new CSRF token for a session
func (s *CSRFTokenStore) GenerateToken(sessionID string) (string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Store token
	s.mu.Lock()
	defer s.mu.Unlock()

	csrfToken := &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(s.ttl),
	}
	s.tokens[sessionID] = csrfToken

	return token, nil
}

// ValidateToken validates a CSRF token for a session
func (s *CSRFTokenStore) ValidateToken(sessionID, token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	csrfToken, exists := s.tokens[sessionID]
	if !exists {
		return false
	}

	// Check expiration
	if time.Now().After(csrfToken.ExpiresAt) {
		return false
	}

	// Compare tokens
	return csrfToken.Token == token
}

// GetToken retrieves the current token for a session
func (s *CSRFTokenStore) GetToken(sessionID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	csrfToken, exists := s.tokens[sessionID]
	if !exists {
		return "", false
	}

	// Check expiration
	if time.Now().After(csrfToken.ExpiresAt) {
		return "", false
	}

	return csrfToken.Token, true
}

// DeleteToken removes a token for a session
func (s *CSRFTokenStore) DeleteToken(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, sessionID)
}

// cleanupExpired removes expired tokens periodically
func (s *CSRFTokenStore) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for sessionID, token := range s.tokens {
			if now.After(token.ExpiresAt) {
				delete(s.tokens, sessionID)
			}
		}
		s.mu.Unlock()
	}
}
