package service

import (
	"errors"
	"fmt"
	"spellingclash/internal/credentials"
	"spellingclash/internal/models"
	"spellingclash/internal/repository"
)

var (
	ErrTeacherRequired = errors.New("teacher account required")
	ErrTeacherKidLink  = errors.New("teacher is not linked to this child")
)

// TeacherService handles teacher-to-child workflows.
type TeacherService struct {
	userRepo       *repository.UserRepository
	familyRepo     *repository.FamilyRepository
	kidRepo        *repository.KidRepository
	teacherKidsRepo *repository.TeacherKidRepository
}

// NewTeacherService creates a new teacher service.
func NewTeacherService(userRepo *repository.UserRepository, familyRepo *repository.FamilyRepository, kidRepo *repository.KidRepository, teacherKidsRepo *repository.TeacherKidRepository) *TeacherService {
	return &TeacherService{
		userRepo:        userRepo,
		familyRepo:      familyRepo,
		kidRepo:         kidRepo,
		teacherKidsRepo: teacherKidsRepo,
	}
}

// VerifyTeacher ensures the provided user is a teacher account.
func (s *TeacherService) VerifyTeacher(userID int64) error {
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || !user.IsTeacher {
		return ErrTeacherRequired
	}
	return nil
}

// GetTeacherKids returns all children linked to a teacher.
func (s *TeacherService) GetTeacherKids(teacherUserID int64) ([]models.Kid, error) {
	if err := s.VerifyTeacher(teacherUserID); err != nil {
		return nil, err
	}
	kids, err := s.teacherKidsRepo.GetTeacherKids(teacherUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher kids: %w", err)
	}
	return kids, nil
}

// CreateTeacherKid creates one child account and links it to the teacher.
func (s *TeacherService) CreateTeacherKid(teacherUserID int64, name, avatarColor string) (*models.Kid, error) {
	if err := s.VerifyTeacher(teacherUserID); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, errors.New("child name is required")
	}
	if avatarColor == "" {
		avatarColor = "#4A90E2"
	}

	username, err := s.generateUniqueUsername()
	if err != nil {
		return nil, err
	}
	password, err := credentials.GenerateKidPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	family, err := s.familyRepo.CreateStandaloneFamily()
	if err != nil {
		return nil, fmt.Errorf("failed to create family container: %w", err)
	}

	kid, err := s.kidRepo.CreateKid(family.FamilyCode, name, username, password, avatarColor)
	if err != nil {
		return nil, fmt.Errorf("failed to create child: %w", err)
	}

	if err := s.teacherKidsRepo.LinkTeacherToKid(teacherUserID, kid.ID); err != nil {
		return nil, err
	}

	return kid, nil
}

// BulkCreateTeacherKids creates multiple children and links them to the teacher.
func (s *TeacherService) BulkCreateTeacherKids(teacherUserID int64, names []string, avatarColor string) ([]models.Kid, error) {
	if err := s.VerifyTeacher(teacherUserID); err != nil {
		return nil, err
	}

	var created []models.Kid
	for _, name := range names {
		kid, err := s.CreateTeacherKid(teacherUserID, name, avatarColor)
		if err != nil {
			return nil, err
		}
		created = append(created, *kid)
	}

	return created, nil
}

// LinkExistingKidByUsername links a teacher to an existing child account.
func (s *TeacherService) LinkExistingKidByUsername(teacherUserID int64, username string) (*models.Kid, error) {
	if err := s.VerifyTeacher(teacherUserID); err != nil {
		return nil, err
	}
	if username == "" {
		return nil, errors.New("child username is required")
	}

	kid, err := s.kidRepo.GetKidByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup child by username: %w", err)
	}
	if kid == nil {
		return nil, ErrKidNotFound
	}

	if err := s.teacherKidsRepo.LinkTeacherToKid(teacherUserID, kid.ID); err != nil {
		return nil, err
	}

	return kid, nil
}

// UpdateTeacherKid updates a child that belongs to the teacher.
func (s *TeacherService) UpdateTeacherKid(teacherUserID, kidID int64, name, avatarColor string) error {
	if err := s.VerifyTeacher(teacherUserID); err != nil {
		return err
	}
	linked, err := s.teacherKidsRepo.IsTeacherLinkedToKid(teacherUserID, kidID)
	if err != nil {
		return err
	}
	if !linked {
		return ErrTeacherKidLink
	}
	if name == "" {
		return errors.New("child name is required")
	}
	if avatarColor == "" {
		avatarColor = "#4A90E2"
	}

	if err := s.kidRepo.UpdateKid(kidID, name, avatarColor); err != nil {
		return fmt.Errorf("failed to update child: %w", err)
	}
	return nil
}

// DeleteTeacherKid deletes a child linked to the teacher.
func (s *TeacherService) DeleteTeacherKid(teacherUserID, kidID int64) error {
	if err := s.VerifyTeacher(teacherUserID); err != nil {
		return err
	}
	linked, err := s.teacherKidsRepo.IsTeacherLinkedToKid(teacherUserID, kidID)
	if err != nil {
		return err
	}
	if !linked {
		return ErrTeacherKidLink
	}
	if err := s.kidRepo.DeleteKid(kidID); err != nil {
		return fmt.Errorf("failed to delete child: %w", err)
	}
	return nil
}

func (s *TeacherService) generateUniqueUsername() (string, error) {
	for i := 0; i < 10; i++ {
		username, err := credentials.GenerateKidUsername()
		if err != nil {
			return "", fmt.Errorf("failed to generate username: %w", err)
		}
		existing, err := s.kidRepo.GetKidByUsername(username)
		if err != nil {
			return "", fmt.Errorf("failed to check username uniqueness: %w", err)
		}
		if existing == nil {
			return username, nil
		}
	}
	return "", errors.New("failed to generate a unique username")
}
