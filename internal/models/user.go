package models

import "time"

// User represents a parent account in the system
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Name         string
	OAuthProvider string
	OAuthSubject  string
	IsAdmin      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Session represents an authenticated session
type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// PasswordResetToken represents a token for password reset
type PasswordResetToken struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
	Used      bool
}

// IsExpired checks if the reset token has expired
func (t *PasswordResetToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}
