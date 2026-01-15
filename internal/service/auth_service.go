package service

import (
	"errors"
	"fmt"
	"spellingclash/internal/models"
	"spellingclash/internal/repository"
	"spellingclash/internal/utils"
	"time"
)

var (
	ErrEmailTaken       = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionExpired   = errors.New("session expired")
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

// Register creates a new user account and auto-creates a family
func (s *AuthService) Register(email, password, name string) (*models.User, error) {
	// Validate inputs
	if err := utils.ValidateEmail(email); err != nil {
		return nil, err
	}
	if err := utils.ValidatePassword(password); err != nil {
		return nil, err
	}
	if err := utils.ValidateName(name); err != nil {
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
	passwordHash, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userRepo.CreateUser(email, passwordHash, name)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Auto-create a family for the new user
	_, err = s.familyRepo.CreateFamily(user.ID)
	if err != nil {
		// Log but don't fail registration - family can be created later
		fmt.Printf("Warning: failed to create family for user %d: %v\n", user.ID, err)
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
	if !utils.CheckPassword(password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}

	// Create session
	sessionID := utils.GenerateSessionID()
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
