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

// HangmanHandler handles hangman game HTTP requests
type HangmanHandler struct {
	db          *database.DB
	listService *service.ListService
	templates   *template.Template
}

// NewHangmanHandler creates a new hangman handler
func NewHangmanHandler(db *database.DB, listService *service.ListService, templates *template.Template) *HangmanHandler {
	return &HangmanHandler{
		db:          db,
		listService: listService,
		templates:   templates,
	}
}

// StartHangman starts a new hangman session
func (h *HangmanHandler) StartHangman(w http.ResponseWriter, r *http.Request) {
	log.Printf("StartHangman called: method=%s path=%s", r.Method, r.URL.Path)
	
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		log.Printf("StartHangman: No kid in context")
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	listIDStr := r.PathValue("listId")
	log.Printf("StartHangman: kid=%s listID=%s", kid.Name, listIDStr)
	listID, err := strconv.ParseInt(listIDStr, 10, 64)
	if err != nil {
		log.Printf("StartHangman: Invalid list ID: %v", err)
		http.Error(w, "Invalid list ID", http.StatusBadRequest)
		return
	}

	// Get words from the list
	log.Printf("StartHangman: Getting words for listID=%d kidID=%d", listID, kid.ID)
	words, err := h.listService.GetListWords(listID, kid.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load words", "Error getting list words", err)
		return
	}

	log.Printf("StartHangman: Got %d words", len(words))
	if len(words) == 0 {
		log.Printf("StartHangman: No words in list, redirecting to dashboard")
		http.Redirect(w, r, "/child/dashboard", http.StatusSeeOther)
		return
	}

	// Shuffle words for random order
	rand.Shuffle(len(words), func(i, j int) {
		words[i], words[j] = words[j], words[i]
	})
	log.Printf("StartHangman: Words shuffled for random order")

	// Create hangman session
	log.Printf("StartHangman: Creating hangman session")
	sessionID, err := h.createHangmanSession(kid.ID, listID, len(words))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start game", "Error creating hangman session", err)
		return
	}

	log.Printf("StartHangman: Created session %d", sessionID)
	// Store words in session state
	wordsJSON, _ := json.Marshal(words)
	err = h.saveHangmanState(kid.ID, sessionID, 0, wordsJSON, 0)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start game", "Error saving hangman state", err)
		return
	}

	log.Printf("StartHangman: Redirecting to /child/hangman/play")

	// Start first game
	http.Redirect(w, r, "/child/hangman/play", http.StatusSeeOther)
}

// PlayHangman renders the hangman game page
func (h *HangmanHandler) PlayHangman(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Get current game state (may be nil if no active game)
	state, err := h.getCurrentGameState(kid.ID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		log.Printf("Error getting game state: %v", err)
		http.Redirect(w, r, "/child/dashboard", http.StatusSeeOther)
		return
	}

	// If no active game, start a new one
	if state == nil || state.IsComplete {
		// Check if there are more words
		if state != nil && state.CurrentWordIdx >= state.TotalWords {
			// Session complete
			h.completeHangmanSession(kid.ID)
			http.Redirect(w, r, "/child/hangman/results", http.StatusSeeOther)
			return
		}

		// Start next word
		words, sessionID, currentIdx, pointsSoFar, err := h.getSessionWords(kid.ID)
		if err != nil || len(words) == 0 {
			http.Redirect(w, r, "/child/dashboard", http.StatusSeeOther)
			return
		}

		if currentIdx >= len(words) {
			h.completeHangmanSession(kid.ID)
			http.Redirect(w, r, "/child/hangman/results", http.StatusSeeOther)
			return
		}

		word := words[currentIdx]
		gameID, err := h.createHangmanGame(sessionID, kid.ID, word.ID, word.WordText)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to start game", "Error creating game", err)
			return
		}

		state = &models.HangmanGameState{
			GameID:          gameID,
			Word:            word.WordText,
			MaskedWord:      strings.Repeat("_ ", len(word.WordText)),
			GuessedLetters:  []string{},
			WrongGuesses:    0,
			MaxWrongGuesses: 6,
			IsWon:           false,
			IsLost:          false,
			IsComplete:      false,
			RemainingWords:  len(words) - currentIdx - 1,
			CurrentWordIdx:  currentIdx,
			TotalWords:      len(words),
			PointsSoFar:     pointsSoFar,
		}
	}

	data := HangmanViewData{
		Title:     "Hangman - SpellingClash",
		Kid:       kid,
		GameState: state,
	}

	if err := h.templates.ExecuteTemplate(w, "hangman.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to render page", "Error rendering hangman template", err)
	}
}

// GuessLetter processes a letter guess
func (h *HangmanHandler) GuessLetter(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	letter := strings.ToLower(strings.TrimSpace(r.FormValue("letter")))
	log.Printf("GuessLetter: kid=%s letter=%s", kid.Name, letter)
	if len(letter) != 1 {
		http.Error(w, "Invalid letter", http.StatusBadRequest)
		return
	}

	// Get current game state
	state, err := h.getCurrentGameState(kid.ID)
	if err != nil || state == nil || state.IsComplete {
		log.Printf("GuessLetter: No active game state, redirecting")
		http.Redirect(w, r, "/child/hangman/play", http.StatusSeeOther)
		return
	}

	log.Printf("GuessLetter: Current state - word=%s wrongGuesses=%d guessedLetters=%v", state.Word, state.WrongGuesses, state.GuessedLetters)

	// Check if letter already guessed
	for _, l := range state.GuessedLetters {
		if l == letter {
			// Already guessed, just return current state
			h.renderGameState(w, kid, state)
			return
		}
	}

	// Add letter to guessed letters
	state.GuessedLetters = append(state.GuessedLetters, letter)

	// Check if letter is in word
	wordLower := strings.ToLower(state.Word)
	letterInWord := strings.Contains(wordLower, letter)

	if !letterInWord {
		state.WrongGuesses++
	}

	// Update masked word
	state.MaskedWord = h.getMaskedWord(state.Word, state.GuessedLetters)

	// Check win/loss conditions
	if !strings.Contains(state.MaskedWord, "_") {
		state.IsWon = true
		state.IsComplete = true
		points := h.calculatePoints(state.WrongGuesses)
		state.PointsSoFar += points
		h.updateGameResult(state.GameID, true, points)
		h.updateSessionPoints(kid.ID, points, true)
	} else if state.WrongGuesses >= state.MaxWrongGuesses {
		state.IsLost = true
		state.IsComplete = true
		h.updateGameResult(state.GameID, false, 0)
		h.updateSessionPoints(kid.ID, 0, false)
	}

	// Save game state
	h.saveGameState(state.GameID, state.GuessedLetters, state.WrongGuesses, state.IsWon, state.IsLost)

	log.Printf("GuessLetter: After save - wrongGuesses=%d guessedLetters=%v isWon=%v isLost=%v", state.WrongGuesses, state.GuessedLetters, state.IsWon, state.IsLost)

	// Render updated game state
	h.renderGameState(w, kid, state)
}

// NextWord moves to the next word
func (h *HangmanHandler) NextWord(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Get session and increment word index
	words, sessionID, currentIdx, pointsSoFar, err := h.getSessionWords(kid.ID)
	if err != nil {
		http.Redirect(w, r, "/child/dashboard", http.StatusSeeOther)
		return
	}

	currentIdx++
	wordsJSON, _ := json.Marshal(words)
	h.saveHangmanState(kid.ID, sessionID, currentIdx, wordsJSON, pointsSoFar)

	http.Redirect(w, r, "/child/hangman/play", http.StatusSeeOther)
}

// ExitGame completes the session and redirects to dashboard
func (h *HangmanHandler) ExitGame(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Complete the session to save points
	h.completeHangmanSession(kid.ID)

	// Redirect to dashboard
	http.Redirect(w, r, "/child/dashboard", http.StatusSeeOther)
}

// ShowResults displays the hangman session results
func (h *HangmanHandler) ShowResults(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Get session results
	results, err := h.getSessionResults(kid.ID)
	if err != nil {
		log.Printf("Error getting results: %v", err)
		http.Redirect(w, r, "/child/dashboard", http.StatusSeeOther)
		return
	}

	// Clean up session state
	h.deleteHangmanState(kid.ID)

	data := HangmanResultsViewData{
		Title:   "Hangman Results - SpellingClash",
		Kid:     kid,
		Results: results,
	}

	if err := h.templates.ExecuteTemplate(w, "hangman_results.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to render page", "Error rendering hangman results template", err)
	}
}

// Helper functions

func (h *HangmanHandler) getMaskedWord(word string, guessedLetters []string) string {
	masked := ""
	wordLower := strings.ToLower(word)
	for _, char := range wordLower {
		if char == ' ' {
			masked += "  "
			continue
		}
		found := false
		for _, letter := range guessedLetters {
			if string(char) == letter {
				found = true
				break
			}
		}
		if found {
			masked += string(char) + " "
		} else {
			masked += "_ "
		}
	}
	return strings.TrimSpace(masked)
}

func (h *HangmanHandler) calculatePoints(wrongGuesses int) int {
	basePoints := 100
	penalty := wrongGuesses * 10
	points := basePoints - penalty
	if points < 10 {
		return 10
	}
	return points
}

func (h *HangmanHandler) renderGameState(w http.ResponseWriter, kid *models.Kid, state *models.HangmanGameState) {
	data := HangmanGameStateViewData{
		Kid:       kid,
		GameState: state,
	}

	if err := h.templates.ExecuteTemplate(w, "hangman_game_state.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to render game state", "Error rendering hangman game state", err)
	}
}

// Database functions

func (h *HangmanHandler) createHangmanSession(kidID, listID int64, totalWords int) (int64, error) {
	query := `INSERT INTO hangman_sessions (kid_id, spelling_list_id, started_at, total_games, games_won, total_points)
			  VALUES (?, ?, ?, ?, 0, 0)`
	return h.db.ExecReturningID(query, kidID, listID, time.Now(), totalWords)
}

func (h *HangmanHandler) createHangmanGame(sessionID, kidID, wordID int64, word string) (int64, error) {
	query := `INSERT INTO hangman_games (session_id, kid_id, word_id, word, guessed_letters, wrong_guesses, max_wrong_guesses, started_at)
			  VALUES (?, ?, ?, ?, ?, 0, 6, ?)`
	return h.db.ExecReturningID(query, sessionID, kidID, wordID, word, "[]", time.Now())
}

func (h *HangmanHandler) saveGameState(gameID int64, guessedLetters []string, wrongGuesses int, isWon, isLost bool) error {
	lettersJSON, _ := json.Marshal(guessedLetters)
	log.Printf("saveGameState: gameID=%d guessedLetters=%v wrongGuesses=%d", gameID, guessedLetters, wrongGuesses)
	query := `UPDATE hangman_games SET guessed_letters = ?, wrong_guesses = ?, is_won = ?, is_lost = ?
			  WHERE id = ?`
	result, err := h.db.Exec(query, string(lettersJSON), wrongGuesses, isWon, isLost, gameID)
	if err != nil {
		log.Printf("saveGameState ERROR: %v", err)
		return err
	}
	rows, _ := result.RowsAffected()
	log.Printf("saveGameState: Updated %d rows", rows)
	return err
}

func (h *HangmanHandler) updateGameResult(gameID int64, isWon bool, points int) error {
	query := `UPDATE hangman_games SET completed_at = ?, is_won = ?, points_earned = ?
			  WHERE id = ?`
	_, err := h.db.Exec(query, time.Now(), isWon, points, gameID)
	return err
}

func (h *HangmanHandler) saveHangmanState(kidID, sessionID int64, currentIdx int, wordsJSON []byte, pointsSoFar int) error {
	// First, try to update existing record
	updateQuery := `UPDATE hangman_state 
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
		insertQuery := `INSERT INTO hangman_state (kid_id, session_id, current_word_idx, words_json, points_so_far)
						VALUES (?, ?, ?, ?, ?)`
		_, err = h.db.Exec(insertQuery, kidID, sessionID, currentIdx, string(wordsJSON), pointsSoFar)
		return err
	}
	
	return nil
}

func (h *HangmanHandler) deleteHangmanState(kidID int64) error {
	query := `DELETE FROM hangman_state WHERE kid_id = ?`
	_, err := h.db.Exec(query, kidID)
	return err
}

func (h *HangmanHandler) getCurrentGameState(kidID int64) (*models.HangmanGameState, error) {
	// Get active game
	query := `SELECT g.id, g.word, g.guessed_letters, g.wrong_guesses, g.max_wrong_guesses, 
			  g.is_won, g.is_lost, s.current_word_idx, s.words_json, s.points_so_far
			  FROM hangman_games g
			  JOIN hangman_state s ON s.session_id = g.session_id
			  WHERE g.kid_id = ? AND g.completed_at IS NULL
			  ORDER BY g.id DESC LIMIT 1`

	var gameID, wrongGuesses, maxWrongGuesses, currentIdx, pointsSoFar int64
	var word, lettersJSON, wordsJSON string
	var isWon, isLost bool

	err := h.db.QueryRow(query, kidID).Scan(&gameID, &word, &lettersJSON, &wrongGuesses,
		&maxWrongGuesses, &isWon, &isLost, &currentIdx, &wordsJSON, &pointsSoFar)

	if err != nil {
		return nil, err
	}

	var guessedLetters []string
	json.Unmarshal([]byte(lettersJSON), &guessedLetters)

	var words []models.Word
	json.Unmarshal([]byte(wordsJSON), &words)

	state := &models.HangmanGameState{
		GameID:          gameID,
		Word:            word,
		MaskedWord:      h.getMaskedWord(word, guessedLetters),
		GuessedLetters:  guessedLetters,
		WrongGuesses:    int(wrongGuesses),
		MaxWrongGuesses: int(maxWrongGuesses),
		IsWon:           isWon,
		IsLost:          isLost,
		IsComplete:      isWon || isLost,
		RemainingWords:  len(words) - int(currentIdx) - 1,
		CurrentWordIdx:  int(currentIdx),
		TotalWords:      len(words),
		PointsSoFar:     int(pointsSoFar),
	}

	return state, nil
}

func (h *HangmanHandler) getSessionWords(kidID int64) ([]models.Word, int64, int, int, error) {
	query := `SELECT session_id, current_word_idx, words_json, points_so_far FROM hangman_state WHERE kid_id = ?`

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

func (h *HangmanHandler) updateSessionPoints(kidID int64, points int, won bool) error {
	// Update session state
	query := `UPDATE hangman_state SET points_so_far = points_so_far + ? WHERE kid_id = ?`
	h.db.Exec(query, points, kidID)

	// Update session totals
	if won {
		query = `UPDATE hangman_sessions 
				 SET games_won = games_won + 1, total_points = total_points + ?
				 WHERE id = (SELECT session_id FROM hangman_state WHERE kid_id = ?)`
	} else {
		query = `UPDATE hangman_sessions 
				 SET total_points = total_points + ?
				 WHERE id = (SELECT session_id FROM hangman_state WHERE kid_id = ?)`
	}
	_, err := h.db.Exec(query, points, kidID)
	return err
}

func (h *HangmanHandler) completeHangmanSession(kidID int64) error {
	query := `UPDATE hangman_sessions SET completed_at = ?
			  WHERE id = (SELECT session_id FROM hangman_state WHERE kid_id = ?)`
	_, err := h.db.Exec(query, time.Now(), kidID)
	return err
}

func (h *HangmanHandler) getSessionResults(kidID int64) (*models.HangmanSession, error) {
	query := `SELECT s.id, s.kid_id, s.spelling_list_id, s.started_at, s.completed_at,
			  s.total_games, s.games_won, s.total_points
			  FROM hangman_sessions s
			  JOIN hangman_state st ON st.session_id = s.id
			  WHERE st.kid_id = ?`

	var session models.HangmanSession
	err := h.db.QueryRow(query, kidID).Scan(&session.ID, &session.KidID,
		&session.SpellingListID, &session.StartedAt, &session.CompletedAt,
		&session.TotalGames, &session.GamesWon, &session.TotalPoints)

	if err != nil {
		return nil, err
	}

	return &session, nil
}
