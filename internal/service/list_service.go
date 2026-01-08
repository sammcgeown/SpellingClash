package service

import (
	"errors"
	"fmt"
	"strings"
	"wordclash/internal/models"
	"wordclash/internal/repository"
)

var (
	ErrListNotFound = errors.New("list not found")
	ErrWordNotFound = errors.New("word not found")
)

// ListService handles spelling list business logic
type ListService struct {
	listRepo   *repository.ListRepository
	familyRepo *repository.FamilyRepository
}

// NewListService creates a new list service
func NewListService(listRepo *repository.ListRepository, familyRepo *repository.FamilyRepository) *ListService {
	return &ListService{
		listRepo:   listRepo,
		familyRepo: familyRepo,
	}
}

// CreateList creates a new spelling list
func (s *ListService) CreateList(familyID, userID int64, name, description string) (*models.SpellingList, error) {
	// Verify user has access to family
	isMember, err := s.familyRepo.IsFamilyMember(userID, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("list name is required")
	}

	// Create list
	list, err := s.listRepo.CreateList(familyID, name, description, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	return list, nil
}

// GetList retrieves a spelling list by ID
func (s *ListService) GetList(listID int64) (*models.SpellingList, error) {
	list, err := s.listRepo.GetListByID(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}
	if list == nil {
		return nil, ErrListNotFound
	}
	return list, nil
}

// GetFamilyLists retrieves all lists for a family
func (s *ListService) GetFamilyLists(familyID, userID int64) ([]models.SpellingList, error) {
	// Verify user has access to family
	isMember, err := s.familyRepo.IsFamilyMember(userID, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	lists, err := s.listRepo.GetFamilyLists(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get family lists: %w", err)
	}

	return lists, nil
}

// GetAllUserLists retrieves all lists from all families a user has access to
func (s *ListService) GetAllUserLists(userID int64) ([]models.SpellingList, error) {
	// Get user's families
	families, err := s.familyRepo.GetUserFamilies(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user families: %w", err)
	}

	// Collect all lists from all families
	var allLists []models.SpellingList
	for _, family := range families {
		lists, err := s.listRepo.GetFamilyLists(family.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get lists for family %d: %w", family.ID, err)
		}
		allLists = append(allLists, lists...)
	}

	return allLists, nil
}

// UpdateList updates a list's name and description
func (s *ListService) UpdateList(listID, userID int64, name, description string) error {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Validate name
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("list name is required")
	}

	// Update list
	if err := s.listRepo.UpdateList(listID, name, description); err != nil {
		return fmt.Errorf("failed to update list: %w", err)
	}

	return nil
}

// DeleteList deletes a spelling list
func (s *ListService) DeleteList(listID, userID int64) error {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Delete list
	if err := s.listRepo.DeleteList(listID); err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
	}

	return nil
}

// AddWord adds a word to a spelling list
func (s *ListService) AddWord(listID, userID int64, wordText string, difficulty int) (*models.Word, error) {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return nil, err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	// Validate word
	wordText = strings.TrimSpace(wordText)
	if wordText == "" {
		return nil, errors.New("word text is required")
	}

	// Validate difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 1
	}

	// Get current word count to determine position
	count, err := s.listRepo.GetWordCount(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get word count: %w", err)
	}

	// Add word at the end
	word, err := s.listRepo.AddWord(listID, wordText, difficulty, count+1)
	if err != nil {
		return nil, fmt.Errorf("failed to add word: %w", err)
	}

	return word, nil
}

// GetListWords retrieves all words for a list
func (s *ListService) GetListWords(listID, userID int64) ([]models.Word, error) {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return nil, err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	words, err := s.listRepo.GetListWords(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get words: %w", err)
	}

	return words, nil
}

// UpdateWord updates a word's text and difficulty
func (s *ListService) UpdateWord(wordID, userID int64, wordText string, difficulty int) error {
	// Get word to get list ID
	word, err := s.listRepo.GetWordByID(wordID)
	if err != nil {
		return fmt.Errorf("failed to get word: %w", err)
	}
	if word == nil {
		return ErrWordNotFound
	}

	// Get list to verify family access
	list, err := s.GetList(word.SpellingListID)
	if err != nil {
		return err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Validate word
	wordText = strings.TrimSpace(wordText)
	if wordText == "" {
		return errors.New("word text is required")
	}

	// Validate difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 1
	}

	// Update word
	if err := s.listRepo.UpdateWord(wordID, wordText, difficulty); err != nil {
		return fmt.Errorf("failed to update word: %w", err)
	}

	return nil
}

// DeleteWord deletes a word from a list
func (s *ListService) DeleteWord(wordID, userID int64) error {
	// Get word to get list ID
	word, err := s.listRepo.GetWordByID(wordID)
	if err != nil {
		return fmt.Errorf("failed to get word: %w", err)
	}
	if word == nil {
		return ErrWordNotFound
	}

	// Get list to verify family access
	list, err := s.GetList(word.SpellingListID)
	if err != nil {
		return err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Delete word
	if err := s.listRepo.DeleteWord(wordID); err != nil {
		return fmt.Errorf("failed to delete word: %w", err)
	}

	return nil
}

// AssignListToKid assigns a spelling list to a kid
func (s *ListService) AssignListToKid(listID, kidID, userID int64) error {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Assign list
	if err := s.listRepo.AssignListToKid(listID, kidID, userID); err != nil {
		return fmt.Errorf("failed to assign list: %w", err)
	}

	return nil
}

// UnassignListFromKid removes a list assignment
func (s *ListService) UnassignListFromKid(listID, kidID, userID int64) error {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return ErrNotFamilyMember
	}

	// Unassign list
	if err := s.listRepo.UnassignListFromKid(listID, kidID); err != nil {
		return fmt.Errorf("failed to unassign list: %w", err)
	}

	return nil
}

// GetKidAssignedLists retrieves all lists assigned to a kid
func (s *ListService) GetKidAssignedLists(kidID int64) ([]models.SpellingList, error) {
	lists, err := s.listRepo.GetKidAssignedLists(kidID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned lists: %w", err)
	}
	return lists, nil
}

// GetListAssignedKids retrieves all kids assigned to a list
func (s *ListService) GetListAssignedKids(listID, userID int64) ([]models.Kid, error) {
	// Get list to verify family access
	list, err := s.GetList(listID)
	if err != nil {
		return nil, err
	}

	// Verify user has access to the list's family
	isMember, err := s.familyRepo.IsFamilyMember(userID, list.FamilyID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify family access: %w", err)
	}
	if !isMember {
		return nil, ErrNotFamilyMember
	}

	kids, err := s.listRepo.GetListAssignedKids(listID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned kids: %w", err)
	}

	return kids, nil
}
