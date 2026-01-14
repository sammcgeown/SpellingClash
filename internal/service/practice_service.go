package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
	"spellingclash/internal/models"
	"spellingclash/internal/repository"
)

// Helper function to convert word IDs to comma-separated string
func wordsToIDString(words []models.Word) string {
	ids := make([]string, len(words))
	for i, word := range words {
		ids[i] = strconv.FormatInt(word.ID, 10)
	}
	return strings.Join(ids, ",")
}

// Helper function to parse word ID string and reorder words
func reorderWordsByIDs(words []models.Word, idString string) []models.Word {
	if idString == "" {
		return words
	}
	
	idStrs := strings.Split(idString, ",")
	idOrder := make([]int64, 0, len(idStrs))
	for _, idStr := range idStrs {
		id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
		if err == nil {
			idOrder = append(idOrder, id)
		}
	}
	
	// Create a map for quick lookup
	wordMap := make(map[int64]models.Word)
	for _, word := range words {
		wordMap[word.ID] = word
	}
	
	// Reorder words according to the ID order
	reordered := make([]models.Word, 0, len(idOrder))
	for _, id := range idOrder {
		if word, exists := wordMap[id]; exists {
			reordered = append(reordered, word)
		}
	}
	
	return reordered
}

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
	allWords, err := s.listRepo.GetListWords(listID)
	if err != nil {
		return nil, nil, err
	}

	if len(allWords) == 0 {
		return nil, nil, errors.New("list has no words")
	}

	var selectedWords []models.Word

	// If list has more than 20 words, select 20 with weighted randomization
	if len(allWords) > 20 {
		selectedWords, err = s.selectWeightedWords(kidID, allWords, 20)
		if err != nil {
			return nil, nil, err
		}
	} else {
		selectedWords = allWords
	}

	// Randomize the order of selected words
	rand.Shuffle(len(selectedWords), func(i, j int) {
		selectedWords[i], selectedWords[j] = selectedWords[j], selectedWords[i]
	})

	// Create practice session
	session, err := s.practiceRepo.CreateSession(kidID, listID, len(selectedWords))
	if err != nil {
		return nil, nil, err
	}

	return session, selectedWords, nil
}

// selectWeightedWords selects words based on performance history
// Words with lower success rates have higher probability of being selected
func (s *PracticeService) selectWeightedWords(kidID int64, words []models.Word, count int) ([]models.Word, error) {
	if count >= len(words) {
		return words, nil
	}

	// Get word IDs
	wordIDs := make([]int64, len(words))
	for i, word := range words {
		wordIDs[i] = word.ID
	}

	// Get performance statistics
	performance, err := s.practiceRepo.GetWordPerformanceForKid(kidID, wordIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get word performance: %w", err)
	}

	// Calculate weights for each word
	// Lower success rate = higher weight
	// Words never attempted get medium-high weight
	type weightedWord struct {
		word   models.Word
		weight float64
	}

	weightedWords := make([]weightedWord, len(words))
	for i, word := range words {
		perf, exists := performance[word.ID]
		var weight float64
		
		if !exists || perf.TotalAttempts == 0 {
			// Never attempted: medium-high weight (0.7)
			weight = 0.7
		} else {
			// Attempted: inverse of success rate
			// 100% success = 0.1 weight
			// 0% success = 1.0 weight
			weight = 1.0 - (perf.SuccessRate * 0.9)
		}
		
		weightedWords[i] = weightedWord{
			word:   word,
			weight: weight,
		}
	}

	// Use weighted random selection
	selected := make([]models.Word, 0, count)
	remaining := make([]weightedWord, len(weightedWords))
	copy(remaining, weightedWords)

	for i := 0; i < count && len(remaining) > 0; i++ {
		// Calculate total weight
		totalWeight := 0.0
		for _, ww := range remaining {
			totalWeight += ww.weight
		}

		// Pick a random point in the total weight
		r := rand.Float64() * totalWeight

		// Find which word corresponds to that point
		cumWeight := 0.0
		selectedIdx := 0
		for idx, ww := range remaining {
			cumWeight += ww.weight
			if r <= cumWeight {
				selectedIdx = idx
				break
			}
		}

		// Add selected word to result
		selected = append(selected, remaining[selectedIdx].word)

		// Remove selected word from remaining
		remaining = append(remaining[:selectedIdx], remaining[selectedIdx+1:]...)
	}

	return selected, nil
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

// GetKidTotalSessionsCount gets the total count of both practice and hangman sessions
func (s *PracticeService) GetKidTotalSessionsCount(kidID int64) (int, error) {
	return s.practiceRepo.GetKidTotalSessionsCount(kidID)
}

// GetKidTotalPoints gets the total points earned by a kid
func (s *PracticeService) GetKidTotalPoints(kidID int64) (int, error) {
	return s.practiceRepo.GetKidTotalPoints(kidID)
}

// SavePracticeState saves the current practice state for a kid
func (s *PracticeService) SavePracticeState(kidID, sessionID int64, currentIndex, correctCount, totalPoints int, startTime time.Time, words []models.Word) error {
	wordOrder := wordsToIDString(words)
	return s.practiceRepo.SavePracticeState(kidID, sessionID, currentIndex, correctCount, totalPoints, startTime, wordOrder)
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

	// Reorder words according to saved order
	words = reorderWordsByIDs(words, state.WordOrder)

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

// GetStrugglingWords gets words a kid is struggling with (below 60% success rate, minimum 3 attempts)
func (s *PracticeService) GetStrugglingWords(kidID int64) ([]repository.StrugglingWord, error) {
	return s.practiceRepo.GetStrugglingWordsForKid(kidID, 0.6, 3)
}

// GetKidStats gets overall statistics for a kid
func (s *PracticeService) GetKidStats(kidID int64) (*models.KidStats, error) {
	return s.practiceRepo.GetKidStats(kidID)
}
