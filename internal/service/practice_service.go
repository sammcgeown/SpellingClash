package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"wordclash/internal/models"
	"wordclash/internal/repository"
)

// PracticeService handles practice game business logic
type PracticeService struct {
	practiceRepo *repository.PracticeRepository
	listRepo     *repository.ListRepository
}

// NewPracticeService creates a new practice service
func NewPracticeService(practiceRepo *repository.PracticeRepository, listRepo *repository.ListRepository) *PracticeService {
	return &PracticeService{
		practiceRepo: practiceRepo,
		listRepo:     listRepo,
	}
}

// StartPracticeSession starts a new practice session for a kid
func (s *PracticeService) StartPracticeSession(kidID, listID int64) (*models.PracticeSession, []models.Word, error) {
	// Get words from the list
	words, err := s.listRepo.GetListWords(listID)
	if err != nil {
		return nil, nil, err
	}

	if len(words) == 0 {
		return nil, nil, errors.New("list has no words")
	}

	// Create practice session
	session, err := s.practiceRepo.CreateSession(kidID, listID, len(words))
	if err != nil {
		return nil, nil, err
	}

	return session, words, nil
}

// CheckAnswer checks if the answer is correct and calculates points
func (s *PracticeService) CheckAnswer(sessionID, wordID int64, answer string, timeTakenMs int, correctWord string, difficulty int) (bool, int, error) {
	// Normalize both strings for comparison (case-insensitive, trim whitespace)
	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer))
	normalizedCorrect := strings.ToLower(strings.TrimSpace(correctWord))

	// Debug logging
	fmt.Printf("DEBUG CheckAnswer: answer='%s' (len=%d), correctWord='%s' (len=%d), normalized answer='%s', normalized correct='%s', match=%v\n", 
		answer, len(answer), correctWord, len(correctWord), normalizedAnswer, normalizedCorrect, normalizedAnswer == normalizedCorrect)

	isCorrect := normalizedAnswer == normalizedCorrect

	// Calculate points
	points := 0
	if isCorrect {
		points = s.calculatePoints(difficulty, timeTakenMs)
	}

	// Record the attempt
	_, err := s.practiceRepo.RecordAttempt(sessionID, wordID, answer, isCorrect, timeTakenMs, points)
	if err != nil {
		return false, 0, err
	}

	return isCorrect, points, nil
}

// calculatePoints calculates points based on difficulty and speed
// Formula: basePoints = difficulty * 10 (10-50 points)
//          speedBonus = max(0, 50 - (timeTakenMs / 100)) up to 50 bonus points
//          totalPoints = basePoints + speedBonus
func (s *PracticeService) calculatePoints(difficulty, timeTakenMs int) int {
	// Base points from difficulty (1-5 scale × 10 = 10-50 points)
	basePoints := difficulty * 10

	// Speed bonus (up to 50 points for very fast answers)
	// Subtract 1 point per 100ms, but don't go below 0
	speedBonus := 50 - (timeTakenMs / 100)
	if speedBonus < 0 {
		speedBonus = 0
	}
	if speedBonus > 50 {
		speedBonus = 50
	}

	return basePoints + speedBonus
}

// CompleteSession marks a session as complete
func (s *PracticeService) CompleteSession(sessionID int64) (*models.PracticeSession, error) {
	// Get all attempts for this session
	attempts, err := s.practiceRepo.GetSessionAttempts(sessionID)
	if err != nil {
		return nil, err
	}

	// Calculate totals
	correctCount := 0
	totalPoints := 0
	for _, attempt := range attempts {
		if attempt.IsCorrect {
			correctCount++
		}
		totalPoints += attempt.PointsEarned
	}

	// Update session
	err = s.practiceRepo.CompleteSession(sessionID, correctCount, totalPoints)
	if err != nil {
		return nil, err
	}

	// Return updated session
	return s.practiceRepo.GetSessionByID(sessionID)
}

// GetSessionResults retrieves session results with attempt details
func (s *PracticeService) GetSessionResults(sessionID int64) (*models.PracticeSession, []models.WordAttempt, error) {
	session, err := s.practiceRepo.GetSessionByID(sessionID)
	if err != nil {
		return nil, nil, err
	}

	attempts, err := s.practiceRepo.GetSessionAttempts(sessionID)
	if err != nil {
		return nil, nil, err
	}

	return session, attempts, nil
}

// GetKidRecentSessions retrieves recent practice sessions for a kid
func (s *PracticeService) GetKidRecentSessions(kidID int64, limit int) ([]models.PracticeSession, error) {
	return s.practiceRepo.GetKidSessions(kidID, limit)
}

// GetKidTotalPoints gets the total points earned by a kid
func (s *PracticeService) GetKidTotalPoints(kidID int64) (int, error) {
	return s.practiceRepo.GetKidTotalPoints(kidID)
}

// SavePracticeState saves the current practice state for a kid
func (s *PracticeService) SavePracticeState(kidID, sessionID int64, currentIndex, correctCount, totalPoints int, startTime time.Time) error {
	return s.practiceRepo.SavePracticeState(kidID, sessionID, currentIndex, correctCount, totalPoints, startTime)
}

// GetPracticeState retrieves the current practice state for a kid and the words
func (s *PracticeService) GetPracticeState(kidID int64) (*models.PracticeState, []models.Word, error) {
	state, err := s.practiceRepo.GetPracticeState(kidID)
	if err != nil {
		return nil, nil, err
	}
	if state == nil {
		return nil, nil, nil
	}

	// Get the session to find the list ID
	session, err := s.practiceRepo.GetSessionByID(state.SessionID)
	if err != nil {
		return nil, nil, err
	}

	// Get words for the list
	words, err := s.listRepo.GetListWords(session.SpellingListID)
	if err != nil {
		return nil, nil, err
	}

	return state, words, nil
}

// DeletePracticeState removes the practice state for a kid
func (s *PracticeService) DeletePracticeState(kidID int64) error {
	return s.practiceRepo.DeletePracticeState(kidID)
}

// SaveWordTiming saves when a word was presented to the kid
func (s *PracticeService) SaveWordTiming(kidID, sessionID int64, wordIndex int, startedAt time.Time) error {
	return s.practiceRepo.SaveWordTiming(kidID, sessionID, wordIndex, startedAt)
}

// GetWordTiming retrieves when a word was presented
func (s *PracticeService) GetWordTiming(kidID, sessionID int64, wordIndex int) (time.Time, error) {
	return s.practiceRepo.GetWordTiming(kidID, sessionID, wordIndex)
}

// DeleteWordTimings removes all word timings for a session
func (s *PracticeService) DeleteWordTimings(kidID, sessionID int64) error {
	return s.practiceRepo.DeleteWordTimings(kidID, sessionID)
}

