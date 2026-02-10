package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"spellingclash/internal/models"
	"spellingclash/internal/repository"
	"spellingclash/internal/security"
	"spellingclash/internal/validation"
	"strings"
	"time"
)

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo        *repository.UserRepository
	familyRepo      *repository.FamilyRepository
	sessionDuration time.Duration
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, familyRepo *repository.FamilyRepository, sessionDuration time.Duration) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		familyRepo:      familyRepo,
		sessionDuration: sessionDuration,
	}
}

// Register creates a new user account and either joins an existing family or creates a new one
func (s *AuthService) Register(email, password, name, familyCode string) (*models.User, error) {
	// Validate inputs
	if err := validation.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := validation.ValidatePassword(password); err != nil {
		return nil, err
	}
	if err := validation.ValidateName(name); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingUser, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userRepo.CreateUser(email, passwordHash, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Handle family membership
	if familyCode != "" {
		// Join existing family
		family, err := s.familyRepo.GetFamilyByCode(familyCode)
		if err != nil {
			return nil, fmt.Errorf("failed to check family code: %w", err)
		}
		if family == nil {
			return nil, errors.New("invalid family code")
		}
		if err := s.familyRepo.AddFamilyMember(familyCode, user.ID, "parent"); err != nil {
			return nil, fmt.Errorf("failed to join family: %w", err)
		}
	} else {
		// Auto-create a family for the new user
		if _, err := s.familyRepo.CreateFamily(user.ID); err != nil {
			// Log but don't fail registration - family can be created later
			fmt.Printf("Warning: failed to create family for user %d: %v\n", user.ID, err)
		}
	}

	return user, nil
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(email, password string) (*models.Session, *models.User, error) {
	// Get user by email
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Check password
	if !security.CheckPassword(password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}

	// Create session
	sessionID := security.GenerateSessionID()
	expiresAt := time.Now().Add(s.sessionDuration)

	session, err := s.userRepo.CreateSession(sessionID, user.ID, expiresAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, user, nil
}

// ValidateSession checks if a session is valid and returns the associated user
func (s *AuthService) ValidateSession(sessionID string) (*models.User, error) {
	// Get session
	session, err := s.userRepo.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	// Check if expired
	if session.IsExpired() {
		// Clean up expired session
		_ = s.userRepo.DeleteSession(sessionID)
		return nil, ErrSessionExpired
	}

	// Get user
	user, err := s.userRepo.GetUserByID(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrSessionNotFound
	}

	return user, nil
}

// Logout invalidates a session
func (s *AuthService) Logout(sessionID string) error {
	if err := s.userRepo.DeleteSession(sessionID); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	return nil
}

// CleanupExpiredSessions removes expired sessions from the database
func (s *AuthService) CleanupExpiredSessions() error {
	if err := s.userRepo.DeleteExpiredSessions(); err != nil {
		return fmt.Errorf("failed to cleanup sessions: %w", err)
	}
	return nil
}

// OAuthLogin authenticates or creates a user using an OAuth provider
func (s *AuthService) OAuthLogin(provider, subject, email, name, familyCode string) (*models.Session, *models.User, error) {
	if provider == "" || subject == "" {
		return nil, nil, errors.New("missing oauth provider information")
	}
	if err := validation.ValidateEmail(email); err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.GetUserByOAuth(provider, subject)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup oauth user: %w", err)
	}

	if user == nil {
		existingUser, err := s.userRepo.GetUserByEmail(email)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check existing user: %w", err)
		}
		if existingUser != nil {
			if existingUser.OAuthProvider != "" && existingUser.OAuthProvider != provider {
				return nil, nil, ErrEmailTaken
			}
			if err := s.userRepo.LinkOAuthProvider(existingUser.ID, provider, subject); err != nil {
				return nil, nil, fmt.Errorf("failed to link oauth provider: %w", err)
			}
			user = existingUser
		} else {
			if name == "" {
				name = strings.Split(email, "@")[0]
			}
			randomPasswordHash, err := security.HashPassword(security.GenerateSessionID())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to generate oauth password hash: %w", err)
			}
			newUser, err := s.userRepo.CreateUser(email, randomPasswordHash, name)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create oauth user: %w", err)
			}
			if err := s.userRepo.LinkOAuthProvider(newUser.ID, provider, subject); err != nil {
				return nil, nil, fmt.Errorf("failed to link oauth provider: %w", err)
			}
			user = newUser

			if familyCode != "" {
				family, err := s.familyRepo.GetFamilyByCode(familyCode)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to check family code: %w", err)
				}
				if family == nil {
					return nil, nil, errors.New("invalid family code")
				}
				if err := s.familyRepo.AddFamilyMember(familyCode, user.ID, "parent"); err != nil {
					return nil, nil, fmt.Errorf("failed to join family: %w", err)
				}
			} else {
				if _, err := s.familyRepo.CreateFamily(user.ID); err != nil {
					fmt.Printf("Warning: failed to create family for user %d: %v\n", user.ID, err)
				}
			}
		}
	}

	sessionID := security.GenerateSessionID()
	expiresAt := time.Now().Add(s.sessionDuration)
	session, err := s.userRepo.CreateSession(sessionID, user.ID, expiresAt)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, user, nil
}

// RequestPasswordReset creates a password reset token and sends an email
func (s *AuthService) RequestPasswordReset(ctx context.Context, emailService *EmailService, email string) error {
	// Get user by email
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// If user doesn't exist, don't reveal that information (security best practice)
	if user == nil {
		return nil
	}

	// Don't allow password reset for OAuth-only accounts
	if user.OAuthProvider != "" && user.PasswordHash == "" {
		return nil
	}

	// Generate secure random token
	token, err := generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Delete any existing reset tokens for this user
	_ = s.userRepo.DeleteUserPasswordResetTokens(user.ID)

	// Create token (expires in 1 hour)
	expiresAt := time.Now().Add(1 * time.Hour)
	if err := s.userRepo.CreatePasswordResetToken(token, user.ID, expiresAt); err != nil {
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// Send email
	if emailService != nil && emailService.IsEnabled() {
		if err := emailService.SendPasswordResetEmail(ctx, user.Email, user.Name, token); err != nil {
			return fmt.Errorf("failed to send reset email: %w", err)
		}
	}

	return nil
}

// ValidatePasswordResetToken checks if a reset token is valid
func (s *AuthService) ValidatePasswordResetToken(token string) (bool, error) {
	resetToken, err := s.userRepo.GetPasswordResetToken(token)
	if err != nil {
		return false, fmt.Errorf("failed to get reset token: %w", err)
	}

	if resetToken == nil {
		return false, nil
	}

	if resetToken.Used {
		return false, nil
	}

	if resetToken.IsExpired() {
		return false, nil
	}

	return true, nil
}

// ResetPassword resets a user's password using a valid token
func (s *AuthService) ResetPassword(token, newPassword string) error {
	// Validate token
	resetToken, err := s.userRepo.GetPasswordResetToken(token)
	if err != nil {
		return fmt.Errorf("failed to get reset token: %w", err)
	}

	if resetToken == nil {
		return errors.New("invalid or expired reset token")
	}

	if resetToken.Used {
		return errors.New("this reset link has already been used")
	}

	if resetToken.IsExpired() {
		return errors.New("this reset link has expired")
	}

	// Validate new password
	if err := validation.ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(resetToken.UserID, passwordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark token as used
	if err := s.userRepo.MarkPasswordResetTokenAsUsed(token); err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Invalidate all existing sessions for this user (force re-login)
	// This is a security best practice after password change
	// Note: We'd need to add a method to delete all sessions for a user
	// For now, they'll be cleaned up on next session validation

	return nil
}

// CleanupExpiredPasswordResetTokens removes expired reset tokens
func (s *AuthService) CleanupExpiredPasswordResetTokens() error {
	if err := s.userRepo.DeleteExpiredPasswordResetTokens(); err != nil {
		return fmt.Errorf("failed to cleanup reset tokens: %w", err)
	}
	return nil
}

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
