package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"spellingclash/internal/database"
	"spellingclash/internal/models"
	"spellingclash/internal/service"
	"strconv"
	"strings"
	"time"
)

// MissingLetterHandler handles missing letter game HTTP requests
type MissingLetterHandler struct {
	db          *database.DB
	listService *service.ListService
	templates   *template.Template
}

// NewMissingLetterHandler creates a new missing letter handler
func NewMissingLetterHandler(db *database.DB, listService *service.ListService, templates *template.Template) *MissingLetterHandler {
	return &MissingLetterHandler{
		db:          db,
		listService: listService,
		templates:   templates,
	}
}

// StartMissingLetter starts a new missing letter session
func (h *MissingLetterHandler) StartMissingLetter(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("listId")
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	// Get words from the list
	words, err := h.listService.GetListWords(listID, kid.ID)
	if err != nil {
		log.Printf("Error getting list words: %v", err)
		http.Error(w, "Failed to load words", http.StatusInternalServerError)
		return
	}

	if len(words) == 0 {
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// Shuffle words for random order
	rand.Shuffle(len(words), func(i, j int) {
		words[i], words[j] = words[j], words[i]
	})

	// Create missing letter session
	sessionID, err := h.createMissingLetterSession(kid.ID, listID, len(words))
	if err != nil {
		log.Printf("Error creating missing letter session: %v", err)
		http.Error(w, "Failed to start game", http.StatusInternalServerError)
		return
	}

	// Store words in session state
	wordsJSON, _ := json.Marshal(words)
	err = h.saveMissingLetterState(kid.ID, sessionID, 0, wordsJSON, 0)
	if err != nil {
		log.Printf("Error saving missing letter state: %v", err)
		http.Error(w, "Failed to start game", http.StatusInternalServerError)
		return
	}

	// Start first game
	http.Redirect(w, r, "/kid/missing-letter/play", http.StatusSeeOther)
}

// PlayMissingLetter renders the missing letter game page
func (h *MissingLetterHandler) PlayMissingLetter(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Try to get current game state
	state, err := h.getCurrentGameState(kid.ID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		log.Printf("Error getting game state: %v", err)
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// If no active game, start a new one
	if state == nil || state.IsComplete {
		// Check if there are more words
		if state != nil && state.CurrentWordIdx >= state.TotalWords {
			// Session complete
			h.completeSession(kid.ID)
			http.Redirect(w, r, "/kid/missing-letter/results", http.StatusSeeOther)
			return
		}

		// Start next word
		words, sessionID, currentIdx, pointsSoFar, err := h.getSessionWords(kid.ID)
		if err != nil || len(words) == 0 {
			http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
			return
		}

		if currentIdx >= len(words) {
			h.completeSession(kid.ID)
			http.Redirect(w, r, "/kid/missing-letter/results", http.StatusSeeOther)
			return
		}

		word := words[currentIdx]

		// Determine missing letter indices based on word length and difficulty
		missingIndices := h.getMissingIndices(word.WordText, word.DifficultyLevel)

		// Create new game
		gameID, err := h.createMissingLetterGame(sessionID, kid.ID, word.ID, word.WordText, missingIndices)
		if err != nil {
			log.Printf("Error creating game: %v", err)
			http.Error(w, "Failed to start game", http.StatusInternalServerError)
			return
		}

		state = &models.MissingLetterGameState{
			GameID:           gameID,
			Word:             word.WordText,
			DisplayWord:      h.getDisplayWord(word.WordText, missingIndices, []string{}),
			MissingIndices:   missingIndices,
			GuessedLetters:   []string{},
			Attempts:         0,
			MaxAttempts:      3,
			IsWon:            false,
			IsLost:           false,
			IsComplete:       false,
			RemainingWords:   len(words) - currentIdx - 1,
			CurrentWordIdx:   currentIdx,
			TotalWords:       len(words),
			PointsSoFar:      pointsSoFar,
			LastGuessCorrect: nil,
		}
	}

	data := map[string]interface{}{
		"Title":     "Missing Letter Mayhem - SpellingClash",
		"Kid":       kid,
		"GameState": state,
	}

	if err := h.templates.ExecuteTemplate(w, "missing_letter.tmpl", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// GuessLetter handles a letter guess
func (h *MissingLetterHandler) GuessLetter(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	missingLettersGuess := strings.ToLower(strings.TrimSpace(r.FormValue("guess")))
	if missingLettersGuess == "" {
		// Empty guess, just return current state
		state, err := h.getCurrentGameState(kid.ID)
		if err != nil {
			http.Redirect(w, r, "/kid/missing-letter/play", http.StatusSeeOther)
			return
		}
		h.renderGameState(w, kid, state)
		return
	}

	// Get current state
	state, err := h.getCurrentGameState(kid.ID)
	if err != nil {
		http.Redirect(w, r, "/kid/missing-letter/play", http.StatusSeeOther)
		return
	}

	// Build the complete word from the guess
	wordLower := strings.ToLower(state.Word)
	guessedWord := h.buildWordFromGuess(state.Word, state.MissingIndices, missingLettersGuess)
	
	state.Attempts++
	state.GuessedLetters = append(state.GuessedLetters, guessedWord)
	
	// Reset bonus flags
	state.LastGuessCorrect = nil
	state.LastValidWordBonus = nil
	
	// Check if the guessed word is correct
	correct := (guessedWord == wordLower)
	
	// Check if it's a valid English word (for bonus points)
	isValidWord := h.isValidWord(guessedWord)
	
	if correct {
		// Win!
		state.LastGuessCorrect = &correct
		state.IsWon = true
		state.IsComplete = true
		points := h.calculatePoints(state.Attempts, len(state.MissingIndices))
		state.PointsSoFar += points
		h.updateGameResult(state.GameID, true, points)
		h.updateSessionPoints(kid.ID, points, true)
	} else if isValidWord && guessedWord != wordLower {
		// Valid word but not the target word - bonus points!
		state.PointsSoFar += 10
		h.updateSessionPoints(kid.ID, 10, false)
		validWordBonus := true
		state.LastValidWordBonus = &validWordBonus
		if state.Attempts >= state.MaxAttempts {
			// Out of attempts
			state.IsLost = true
			state.IsComplete = true
			h.updateGameResult(state.GameID, false, 10)
		}
	} else {
		// Incorrect guess
		incorrectGuess := false
		state.LastGuessCorrect = &incorrectGuess
		if state.Attempts >= state.MaxAttempts {
			// Loss
			state.IsLost = true
			state.IsComplete = true
			h.updateGameResult(state.GameID, false, 0)
			h.updateSessionPoints(kid.ID, 0, false)
		}
	}

	// Update display word (only show correct if won)
	if state.IsWon {
		state.DisplayWord = state.Word
	} else {
		state.DisplayWord = h.getDisplayWord(state.Word, state.MissingIndices, []string{})
	}

	// Save game state
	h.saveGameState(state.GameID, state.GuessedLetters, state.Attempts, state.IsWon, state.IsLost)

	// Render updated game state
	h.renderGameState(w, kid, state)
}

// NextWord moves to the next word
func (h *MissingLetterHandler) NextWord(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get session and increment word index
	words, sessionID, currentIdx, pointsSoFar, err := h.getSessionWords(kid.ID)
	if err != nil {
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	currentIdx++
	wordsJSON, _ := json.Marshal(words)
	h.saveMissingLetterState(kid.ID, sessionID, currentIdx, wordsJSON, pointsSoFar)

	http.Redirect(w, r, "/kid/missing-letter/play", http.StatusSeeOther)
}

// ExitGame completes the session and redirects to dashboard
func (h *MissingLetterHandler) ExitGame(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Complete the session to save points
	h.completeSession(kid.ID)

	// Redirect to dashboard
	http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
}

// ShowResults displays the missing letter session results
func (h *MissingLetterHandler) ShowResults(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get session results
	results, err := h.getSessionResults(kid.ID)
	if err != nil {
		log.Printf("Error getting results: %v", err)
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// Clean up session state
	h.deleteMissingLetterState(kid.ID)

	data := map[string]interface{}{
		"Title":   "Missing Letter Results - SpellingClash",
		"Kid":     kid,
		"Results": results,
	}

	if err := h.templates.ExecuteTemplate(w, "missing_letter_results.tmpl", data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// Helper functions

// getMissingIndices determines which letters should be missing based on word length and difficulty
func (h *MissingLetterHandler) getMissingIndices(word string, difficulty int) []int {
	wordLen := len(word)
	
	// Calculate number of letters to hide based on difficulty and word length
	var numMissing int
	
	switch difficulty {
	case 1: // Easy
		if wordLen <= 4 {
			numMissing = 1
		} else if wordLen <= 6 {
			numMissing = 2
		} else {
			numMissing = 2
		}
	case 2: // Medium-Easy
		if wordLen <= 4 {
			numMissing = 1
		} else if wordLen <= 6 {
			numMissing = 2
		} else {
			numMissing = 3
		}
	case 3: // Medium
		if wordLen <= 4 {
			numMissing = 1
		} else if wordLen <= 6 {
			numMissing = 2
		} else if wordLen <= 8 {
			numMissing = 3
		} else {
			numMissing = 4
		}
	case 4: // Medium-Hard
		if wordLen <= 4 {
			numMissing = 2
		} else if wordLen <= 6 {
			numMissing = 3
		} else if wordLen <= 8 {
			numMissing = 4
		} else {
			numMissing = 5
		}
	case 5: // Hard
		if wordLen <= 4 {
			numMissing = 2
		} else if wordLen <= 6 {
			numMissing = 3
		} else if wordLen <= 8 {
			numMissing = 5
		} else {
			numMissing = 6
		}
	default:
		numMissing = 2
	}

	// Don't hide more than 50% of the word (was 33%)
	maxMissing := wordLen / 2
	if maxMissing < 1 {
		maxMissing = 1
	}
	if numMissing > maxMissing {
		numMissing = maxMissing
	}

	// Randomly select indices to hide
	indices := make([]int, 0)
	availableIndices := make([]int, 0)
	
	// For difficulty 1, avoid first and last letters
	// For difficulty 2-3, avoid just the first letter
	// For difficulty 4-5, any letter can be hidden
	startIdx := 0
	endIdx := wordLen
	
	if difficulty == 1 && wordLen > 3 {
		startIdx = 1
		endIdx = wordLen - 1
	} else if difficulty <= 3 && wordLen > 2 {
		startIdx = 1
	}

	for i := startIdx; i < endIdx; i++ {
		if word[i] != ' ' { // Don't hide spaces
			availableIndices = append(availableIndices, i)
		}
	}

	// Shuffle and pick
	rand.Shuffle(len(availableIndices), func(i, j int) {
		availableIndices[i], availableIndices[j] = availableIndices[j], availableIndices[i]
	})

	for i := 0; i < numMissing && i < len(availableIndices); i++ {
		indices = append(indices, availableIndices[i])
	}

	return indices
}

// getDisplayWord creates the display version with blanks for missing letters
func (h *MissingLetterHandler) getDisplayWord(word string, missingIndices []int, guessedLetters []string) string {
	result := ""
	wordLower := strings.ToLower(word)
	
	// Check if correct guess was made
	correctGuess := false
	if len(guessedLetters) > 0 {
		missingLetters := ""
		for _, idx := range missingIndices {
			if idx < len(wordLower) {
				missingLetters += string(wordLower[idx])
			}
		}
		for _, guess := range guessedLetters {
			if guess == missingLetters {
				correctGuess = true
				break
			}
		}
	}

	// Build display word
	for i, char := range word {
		isMissing := false
		for _, idx := range missingIndices {
			if idx == i {
				isMissing = true
				break
			}
		}
		
		if isMissing && !correctGuess {
			result += "_"
		} else {
			result += string(char)
		}
	}
	
	return result
}

func (h *MissingLetterHandler) calculatePoints(attempts, numMissing int) int {
	basePoints := 50 * numMissing // More missing letters = more points potential
	penalty := (attempts - 1) * 15
	points := basePoints - penalty
	if points < 10 {
		return 10
	}
	return points
}

// buildWordFromGuess constructs a complete word by inserting the guessed letters into the missing positions
func (h *MissingLetterHandler) buildWordFromGuess(word string, missingIndices []int, guess string) string {
	wordLower := strings.ToLower(word)
	result := []rune(wordLower)
	guessRunes := []rune(guess)
	
	// Insert each guessed letter at the corresponding missing index
	for i, idx := range missingIndices {
		if i < len(guessRunes) && idx < len(result) {
			result[idx] = guessRunes[i]
		}
	}
	
	return string(result)
}

// isValidWord checks if a word is a valid English word
// For now, we'll use a simple check against common words
// In production, you might want to use a proper dictionary API or word list
func (h *MissingLetterHandler) isValidWord(word string) bool {
	// Simple implementation: check if word is at least 2 characters
	// and contains only letters
	if len(word) < 2 {
		return false
	}
	
	for _, char := range word {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')) {
			return false
		}
	}
	
	// For now, we'll consider all alphabetic words as valid
	// You could enhance this with a word list check
	return true
}

func (h *MissingLetterHandler) renderGameState(w http.ResponseWriter, kid *models.Kid, state *models.MissingLetterGameState) {
	data := map[string]interface{}{
		"Kid":       kid,
		"GameState": state,
	}

	if err := h.templates.ExecuteTemplate(w, "missing_letter_game_state.tmpl", data); err != nil {
		log.Printf("Error rendering game state: %v", err)
		http.Error(w, "Failed to render game state", http.StatusInternalServerError)
	}
}

// Database functions

func (h *MissingLetterHandler) createMissingLetterSession(kidID, listID int64, totalWords int) (int64, error) {
	query := `INSERT INTO missing_letter_sessions (kid_id, spelling_list_id, started_at, total_games, games_won, total_points)
			  VALUES (?, ?, ?, ?, 0, 0)`
	return h.db.ExecReturningID(query, kidID, listID, time.Now(), totalWords)
}

func (h *MissingLetterHandler) createMissingLetterGame(sessionID, kidID, wordID int64, word string, missingIndices []int) (int64, error) {
	indicesJSON, _ := json.Marshal(missingIndices)
	query := `INSERT INTO missing_letter_games (session_id, kid_id, word_id, word, missing_indices, guessed_letters, attempts, max_attempts, started_at)
			  VALUES (?, ?, ?, ?, ?, ?, 0, 3, ?)`
	return h.db.ExecReturningID(query, sessionID, kidID, wordID, word, string(indicesJSON), "[]", time.Now())
}

func (h *MissingLetterHandler) saveGameState(gameID int64, guessedLetters []string, attempts int, isWon, isLost bool) error {
	lettersJSON, _ := json.Marshal(guessedLetters)
	query := `UPDATE missing_letter_games SET guessed_letters = ?, attempts = ?, is_won = ?, is_lost = ?
			  WHERE id = ?`
	_, err := h.db.Exec(query, string(lettersJSON), attempts, isWon, isLost, gameID)
	return err
}

func (h *MissingLetterHandler) updateGameResult(gameID int64, isWon bool, points int) error {
	query := `UPDATE missing_letter_games SET completed_at = ?, is_won = ?, points_earned = ?
			  WHERE id = ?`
	_, err := h.db.Exec(query, time.Now(), isWon, points, gameID)
	return err
}

func (h *MissingLetterHandler) saveMissingLetterState(kidID, sessionID int64, currentIdx int, wordsJSON []byte, pointsSoFar int) error {
	// First, try to update existing record
	updateQuery := `UPDATE missing_letter_state 
					SET session_id = ?, current_word_idx = ?, words_json = ?, points_so_far = ?, updated_at = CURRENT_TIMESTAMP
					WHERE kid_id = ?`
	result, err := h.db.Exec(updateQuery, sessionID, currentIdx, string(wordsJSON), pointsSoFar, kidID)
	if err != nil {
		return err
	}
	
	// Check if any rows were updated
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	// If no rows were updated, insert a new record
	if rowsAffected == 0 {
		insertQuery := `INSERT INTO missing_letter_state (kid_id, session_id, current_word_idx, words_json, points_so_far)
						VALUES (?, ?, ?, ?, ?)`
		_, err = h.db.Exec(insertQuery, kidID, sessionID, currentIdx, string(wordsJSON), pointsSoFar)
		return err
	}
	
	return nil
}

func (h *MissingLetterHandler) deleteMissingLetterState(kidID int64) error {
	query := `DELETE FROM missing_letter_state WHERE kid_id = ?`
	_, err := h.db.Exec(query, kidID)
	return err
}

func (h *MissingLetterHandler) getCurrentGameState(kidID int64) (*models.MissingLetterGameState, error) {
	query := `SELECT g.id, g.word, g.missing_indices, g.guessed_letters, g.attempts, g.max_attempts, 
			  g.is_won, g.is_lost, s.current_word_idx, s.words_json, s.points_so_far
			  FROM missing_letter_games g
			  JOIN missing_letter_state s ON s.session_id = g.session_id
			  WHERE g.kid_id = ? AND g.completed_at IS NULL
			  ORDER BY g.id DESC LIMIT 1`

	var gameID, attempts, maxAttempts, currentIdx, pointsSoFar int64
	var word, indicesJSON, lettersJSON, wordsJSON string
	var isWon, isLost bool

	err := h.db.QueryRow(query, kidID).Scan(&gameID, &word, &indicesJSON, &lettersJSON, &attempts,
		&maxAttempts, &isWon, &isLost, &currentIdx, &wordsJSON, &pointsSoFar)

	if err != nil {
		return nil, err
	}

	var missingIndices []int
	var guessedLetters []string
	json.Unmarshal([]byte(indicesJSON), &missingIndices)
	json.Unmarshal([]byte(lettersJSON), &guessedLetters)

	var words []models.Word
	json.Unmarshal([]byte(wordsJSON), &words)

	// Determine last guess correctness
	var lastGuessCorrect *bool
	if len(guessedLetters) > 0 {
		wordLower := strings.ToLower(word)
		missingLetters := ""
		for _, idx := range missingIndices {
			if idx < len(wordLower) {
				missingLetters += string(wordLower[idx])
			}
		}
		lastGuess := guessedLetters[len(guessedLetters)-1]
		correct := (lastGuess == missingLetters)
		lastGuessCorrect = &correct
	}

	state := &models.MissingLetterGameState{
		GameID:           gameID,
		Word:             word,
		DisplayWord:      h.getDisplayWord(word, missingIndices, guessedLetters),
		MissingIndices:   missingIndices,
		GuessedLetters:   guessedLetters,
		Attempts:         int(attempts),
		MaxAttempts:      int(maxAttempts),
		IsWon:            isWon,
		IsLost:           isLost,
		IsComplete:       isWon || isLost,
		RemainingWords:   len(words) - int(currentIdx) - 1,
		CurrentWordIdx:   int(currentIdx),
		TotalWords:       len(words),
		PointsSoFar:      int(pointsSoFar),
		LastGuessCorrect: lastGuessCorrect,
	}

	return state, nil
}

func (h *MissingLetterHandler) getSessionWords(kidID int64) ([]models.Word, int64, int, int, error) {
	query := `SELECT session_id, current_word_idx, words_json, points_so_far FROM missing_letter_state WHERE kid_id = ?`

	var sessionID int64
	var currentIdx, pointsSoFar int
	var wordsJSON string

	err := h.db.QueryRow(query, kidID).Scan(&sessionID, &currentIdx, &wordsJSON, &pointsSoFar)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	var words []models.Word
	json.Unmarshal([]byte(wordsJSON), &words)

	return words, sessionID, currentIdx, pointsSoFar, nil
}

func (h *MissingLetterHandler) updateSessionPoints(kidID int64, points int, won bool) error {
	// Update session state
	query := `UPDATE missing_letter_state SET points_so_far = points_so_far + ? WHERE kid_id = ?`
	h.db.Exec(query, points, kidID)

	// Update session totals
	if won {
		query = `UPDATE missing_letter_sessions 
				 SET games_won = games_won + 1, total_points = total_points + ?
				 WHERE id = (SELECT session_id FROM missing_letter_state WHERE kid_id = ?)`
	} else {
		query = `UPDATE missing_letter_sessions 
				 SET total_points = total_points + ?
				 WHERE id = (SELECT session_id FROM missing_letter_state WHERE kid_id = ?)`
	}
	_, err := h.db.Exec(query, points, kidID)
	return err
}

func (h *MissingLetterHandler) completeSession(kidID int64) error {
	query := `UPDATE missing_letter_sessions SET completed_at = ?
			  WHERE id = (SELECT session_id FROM missing_letter_state WHERE kid_id = ?)`
	_, err := h.db.Exec(query, time.Now(), kidID)
	return err
}

func (h *MissingLetterHandler) getSessionResults(kidID int64) (*models.MissingLetterSession, error) {
	query := `SELECT s.id, s.kid_id, s.spelling_list_id, s.started_at, s.completed_at,
			  s.total_games, s.games_won, s.total_points
			  FROM missing_letter_sessions s
			  JOIN missing_letter_state st ON st.session_id = s.id
			  WHERE st.kid_id = ?`

	var session models.MissingLetterSession
	err := h.db.QueryRow(query, kidID).Scan(&session.ID, &session.KidID,
		&session.SpellingListID, &session.StartedAt, &session.CompletedAt,
		&session.TotalGames, &session.GamesWon, &session.TotalPoints)

	if err != nil {
		return nil, err
	}

	return &session, nil
}
