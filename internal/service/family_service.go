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
	ErrFamilyNotFound     = errors.New("family not found")
	ErrNotFamilyMember    = errors.New("user is not a member of this family")
	ErrKidNotFound        = errors.New("kid not found")
	ErrInvalidAvatarColor = errors.New("invalid avatar color")
)

// FamilyService handles family and kid business logic
type FamilyService struct {
	familyRepo *repository.FamilyRepository
	kidRepo    *repository.KidRepository
}

// NewFamilyService creates a new family service
func NewFamilyService(familyRepo *repository.FamilyRepository, kidRepo *repository.KidRepository) *FamilyService {
	return &FamilyService{
		familyRepo: familyRepo,
		kidRepo:    kidRepo,
	}
}

// CreateFamily creates a new family with the user as admin
func (s *FamilyService) CreateFamily(name string, creatorUserID int64) (*models.Family, error) {
	if name == "" {
		return nil, errors.New("family name is required")
	}

	family, err := s.familyRepo.CreateFamily(name, creatorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create family: %w", err)
	}

	return family, nil
}

// GetUserFamilies retrieves all families a user belongs to
func (s *FamilyService) GetUserFamilies(userID int64) ([]models.Family, error) {
	families, err := s.familyRepo.GetUserFamilies(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user families: %w", err)
	}
	return families, nil
}

// GetFamily retrieves a family by ID
func (s *FamilyService) GetFamily(familyID int64) (*models.Family, error) {
	family, err := s.familyRepo.GetFamilyByID(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get family: %w", err)
	}
	if family == nil {
		return nil, ErrFamilyNotFound
	}
	return family, nil
}

// VerifyFamilyAccess checks if a user has access to a family
func (s *FamilyService) VerifyFamilyAccess(userID, familyID int64) error {
	isMember, err := s.familyRepo.IsFamilyMember(userID, familyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}
	return nil
}

// AddFamilyMember adds a user to a family
func (s *FamilyService) AddFamilyMember(familyID, inviterUserID, newUserID int64) error {
	// Verify inviter has access
	if err := s.VerifyFamilyAccess(inviterUserID, familyID); err != nil {
		return err
	}

	// Add the new member
	if err := s.familyRepo.AddFamilyMember(familyID, newUserID, "parent"); err != nil {
		return fmt.Errorf("failed to add family member: %w", err)
	}

	return nil
}

// GetFamilyMembers retrieves all members of a family
func (s *FamilyService) GetFamilyMembers(familyID int64) ([]models.FamilyMember, []models.User, error) {
	members, users, err := s.familyRepo.GetFamilyMembers(familyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get family members: %w", err)
	}
	return members, users, nil
}

// CreateKid creates a new kid profile in a family
func (s *FamilyService) CreateKid(familyID, creatorUserID int64, name, avatarColor string) (*models.Kid, error) {
	// Verify user has access to family
	if err := s.VerifyFamilyAccess(creatorUserID, familyID); err != nil {
		return nil, err
	}

	// Validate inputs
	if name == "" {
		return nil, errors.New("kid name is required")
	}

	// Use default color if not provided
	if avatarColor == "" {
		avatarColor = "#4A90E2"
	}

	// Generate random username and password
	username, err := utils.GenerateKidUsername()
	if err != nil {
		return nil, fmt.Errorf("failed to generate username: %w", err)
	}

	password, err := utils.GenerateKidPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	// Ensure username is unique (retry if collision)
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		existing, err := s.kidRepo.GetKidByUsername(username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
		}
		if existing == nil {
			break // Username is unique
		}
		// Generate a new username
		username, err = utils.GenerateKidUsername()
		if err != nil {
			return nil, fmt.Errorf("failed to generate username: %w", err)
		}
	}

	// Create kid
	kid, err := s.kidRepo.CreateKid(familyID, name, username, password, avatarColor)
	if err != nil {
		return nil, fmt.Errorf("failed to create kid: %w", err)
	}

	return kid, nil
}

// GetFamilyKids retrieves all kids in a family
func (s *FamilyService) GetFamilyKids(familyID, userID int64) ([]models.Kid, error) {
	// Verify user has access to family
	if err := s.VerifyFamilyAccess(userID, familyID); err != nil {
		return nil, err
	}

	kids, err := s.kidRepo.GetFamilyKids(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get family kids: %w", err)
	}

	return kids, nil
}

// GetKid retrieves a kid by ID
func (s *FamilyService) GetKid(kidID int64) (*models.Kid, error) {
	kid, err := s.kidRepo.GetKidByID(kidID)
	if err != nil {
		return nil, fmt.Errorf("failed to get kid: %w", err)
	}
	if kid == nil {
		return nil, ErrKidNotFound
	}
	return kid, nil
}

// GetKidByUsername retrieves a kid by username
func (s *FamilyService) GetKidByUsername(username string) (*models.Kid, error) {
	kid, err := s.kidRepo.GetKidByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get kid by username: %w", err)
	}
	return kid, nil
}

// GetAllKids retrieves all kids from all families
func (s *FamilyService) GetAllKids() ([]models.Kid, error) {
	kids, err := s.kidRepo.GetAllKids()
	if err != nil {
		return nil, fmt.Errorf("failed to get all kids: %w", err)
	}
	return kids, nil
}

// UpdateKid updates a kid's information
func (s *FamilyService) UpdateKid(kidID, userID int64, name, avatarColor string) error {
	// Get kid to verify family access
	kid, err := s.GetKid(kidID)
	if err != nil {
		return err
	}

	// Verify user has access to the kid's family
	if err := s.VerifyFamilyAccess(userID, kid.FamilyID); err != nil {
		return err
	}

	// Validate inputs
	if name == "" {
		return errors.New("kid name is required")
	}

	// Update kid
	if err := s.kidRepo.UpdateKid(kidID, name, avatarColor); err != nil {
		return fmt.Errorf("failed to update kid: %w", err)
	}

	return nil
}

// RegenerateKidPassword generates a new random password for a kid
func (s *FamilyService) RegenerateKidPassword(kidID, userID int64) (string, error) {
	// Get kid to verify family access
	kid, err := s.GetKid(kidID)
	if err != nil {
		return "", err
	}

	// Verify user has access to the kid's family
	if err := s.VerifyFamilyAccess(userID, kid.FamilyID); err != nil {
		return "", err
	}

	// Generate new password
	newPassword, err := utils.GenerateKidPassword()
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	// Update password
	if err := s.kidRepo.UpdateKidPassword(kidID, newPassword); err != nil {
		return "", fmt.Errorf("failed to update kid password: %w", err)
	}

	return newPassword, nil
}

// DeleteKid deletes a kid profile
func (s *FamilyService) DeleteKid(kidID, userID int64) error {
	// Get kid to verify family access
	kid, err := s.GetKid(kidID)
	if err != nil {
		return err
	}

	// Verify user has access to the kid's family
	if err := s.VerifyFamilyAccess(userID, kid.FamilyID); err != nil {
		return err
	}

	// Delete kid
	if err := s.kidRepo.DeleteKid(kidID); err != nil {
		return fmt.Errorf("failed to delete kid: %w", err)
	}

	return nil
}

// GetAllUserKids retrieves all kids from all families a user has access to
func (s *FamilyService) GetAllUserKids(userID int64) ([]models.Kid, error) {
	// Get user's families
	families, err := s.GetUserFamilies(userID)
	if err != nil {
		return nil, err
	}

	// Collect all kids from all families
	var allKids []models.Kid
	for _, family := range families {
		kids, err := s.kidRepo.GetFamilyKids(family.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get kids for family %d: %w", family.ID, err)
		}
		allKids = append(allKids, kids...)
	}

	return allKids, nil
}

// CreateKidSession creates a new session for a kid
func (s *FamilyService) CreateKidSession(kidID int64) (string, time.Time, error) {
	// Verify kid exists
	kid, err := s.GetKid(kidID)
	if err != nil {
		return "", time.Time{}, err
	}
	if kid == nil {
		return "", time.Time{}, ErrKidNotFound
	}

	// Generate session ID
	sessionID := utils.GenerateSessionID()
	expiresAt := time.Now().Add(24 * time.Hour) // Kid sessions last 24 hours

	// Create session in database
	if err := s.kidRepo.CreateKidSession(sessionID, kidID, expiresAt); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create kid session: %w", err)
	}

	return sessionID, expiresAt, nil
}

// ValidateKidSession validates a kid session and returns the kid ID
func (s *FamilyService) ValidateKidSession(sessionID string) (int64, error) {
	kidID, err := s.kidRepo.GetKidSession(sessionID)
	if err != nil {
		return 0, fmt.Errorf("invalid kid session: %w", err)
	}
	return kidID, nil
}

// LogoutKid removes a kid session
func (s *FamilyService) LogoutKid(sessionID string) error {
	if err := s.kidRepo.DeleteKidSession(sessionID); err != nil {
		return fmt.Errorf("failed to logout kid: %w", err)
	}
	return nil
}

// CleanupExpiredKidSessions removes expired kid sessions
func (s *FamilyService) CleanupExpiredKidSessions() error {
	if err := s.kidRepo.DeleteExpiredKidSessions(); err != nil {
		return fmt.Errorf("failed to cleanup kid sessions: %w", err)
	}
	return nil
}

// JoinFamilyByCode allows a user to join a family using its code
func (s *FamilyService) JoinFamilyByCode(userID int64, familyCode string) error {
	if familyCode == "" {
		return errors.New("family code is required")
	}

	// Find family by code
	family, err := s.familyRepo.GetFamilyByCode(familyCode)
	if err != nil {
		return fmt.Errorf("failed to find family: %w", err)
	}
	if family == nil {
		return errors.New("invalid family code")
	}

	// Check if already a member
	isMember, err := s.familyRepo.IsFamilyMember(userID, family.ID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if isMember {
		return errors.New("you are already a member of this family")
	}

	// Add as member
	if err := s.familyRepo.AddFamilyMember(family.ID, userID, "parent"); err != nil {
		return fmt.Errorf("failed to join family: %w", err)
	}

	return nil
}

// LeaveFamily allows a user to leave a family
func (s *FamilyService) LeaveFamily(userID, familyID int64) error {
	// Verify user is a member
	isMember, err := s.familyRepo.IsFamilyMember(userID, familyID)
	if err != nil {
		return fmt.Errorf("failed to verify membership: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Remove from family
	if err := s.familyRepo.RemoveUserFromFamily(userID, familyID); err != nil {
		return fmt.Errorf("failed to leave family: %w", err)
	}

	return nil
}
