package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
	"wordclash/internal/service"
)

// PracticeHandler handles practice game HTTP requests
type PracticeHandler struct {
	practiceService *service.PracticeService
	listService     *service.ListService
	templates       *template.Template
}

// NewPracticeHandler creates a new practice handler
func NewPracticeHandler(practiceService *service.PracticeService, listService *service.ListService, templates *template.Template) *PracticeHandler {
	return &PracticeHandler{
		practiceService: practiceService,
		listService:     listService,
		templates:       templates,
	}
}

// StartPractice starts a new practice session
func (h *PracticeHandler) StartPractice(w http.ResponseWriter, r *http.Request) {
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

	// Start practice session
	session, words, err := h.practiceService.StartPracticeSession(kid.ID, listID)
	if err != nil {
		log.Printf("Error starting practice session: %v", err)
		http.Error(w, "Failed to start practice session", http.StatusInternalServerError)
		return
	}

	// Save practice state to database
	if err := h.practiceService.SavePracticeState(kid.ID, session.ID, 0, 0, 0, time.Now()); err != nil {
		log.Printf("Error saving practice state: %v", err)
		http.Error(w, "Failed to save practice state", http.StatusInternalServerError)
		return
	}

	// Save initial word timing (first word)
	if len(words) > 0 {
		if err := h.practiceService.SaveWordTiming(kid.ID, session.ID, 0, time.Now()); err != nil {
			log.Printf("Error saving word timing: %v", err)
		}
	}

	// Redirect to practice page
	http.Redirect(w, r, "/kid/practice", http.StatusSeeOther)
}

// ShowPractice displays the practice game interface
func (h *PracticeHandler) ShowPractice(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
		return
	}

	// Get practice state from database
	state, words, err := h.practiceService.GetPracticeState(kid.ID)
	if err != nil || state == nil {
		// No active session
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// Check if session is complete
	if state.CurrentIndex >= len(words) {
		http.Redirect(w, r, "/kid/practice/results", http.StatusSeeOther)
		return
	}

	// Record word start time if not already recorded
	wordTiming, err := h.practiceService.GetWordTiming(kid.ID, state.SessionID, state.CurrentIndex)
	if err != nil {
		// Save the timing for this word
		if err := h.practiceService.SaveWordTiming(kid.ID, state.SessionID, state.CurrentIndex, time.Now()); err != nil {
			log.Printf("Error saving word timing: %v", err)
		}
		wordTiming = time.Now()
	}

	currentWord := words[state.CurrentIndex]

	data := map[string]interface{}{
		"Title":        "Practice - WordClash",
		"Kid":          kid,
		"Word":         currentWord,
		"CurrentIndex": state.CurrentIndex + 1,
		"TotalWords":   len(words),
		"CorrectCount": state.CorrectCount,
		"TotalPoints":  state.TotalPoints,
		"WordTiming":   wordTiming,
	}

	if err := h.templates.ExecuteTemplate(w, "practice.tmpl", data); err != nil {
		log.Printf("Error rendering practice template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// SubmitAnswer handles answer submission
func (h *PracticeHandler) SubmitAnswer(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get practice state from database
	state, words, err := h.practiceService.GetPracticeState(kid.ID)
	if err != nil || state == nil {
		http.Error(w, "No active practice session", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	answer := r.FormValue("answer")
	currentWord := words[state.CurrentIndex]

	// Log for debugging
	log.Printf("Kid %d answering word '%s' with '%s'", kid.ID, currentWord.WordText, answer)

	// Get word start time
	startTime, err := h.practiceService.GetWordTiming(kid.ID, state.SessionID, state.CurrentIndex)
	if err != nil {
		startTime = time.Now()
	}
	timeTakenMs := int(time.Since(startTime).Milliseconds())

	// Check answer
	isCorrect, points, err := h.practiceService.CheckAnswer(
		state.SessionID,
		currentWord.ID,
		answer,
		timeTakenMs,
		currentWord.WordText,
		currentWord.DifficultyLevel,
	)
	if err != nil {
		log.Printf("Error checking answer: %v", err)
		http.Error(w, "Failed to check answer", http.StatusInternalServerError)
		return
	}

	// Update state
	newCorrectCount := state.CorrectCount
	if isCorrect {
		newCorrectCount++
	}
	newTotalPoints := state.TotalPoints + points
	newIndex := state.CurrentIndex + 1

	// Save updated state to database
	if err := h.practiceService.SavePracticeState(kid.ID, state.SessionID, newIndex, newCorrectCount, newTotalPoints, state.StartTime); err != nil {
		log.Printf("Error saving practice state: %v", err)
		http.Error(w, "Failed to save state", http.StatusInternalServerError)
		return
	}

	// Check if session is complete
	if newIndex >= len(words) {
		// Complete the session
		_, err := h.practiceService.CompleteSession(state.SessionID)
		if err != nil {
			log.Printf("Error completing session: %v", err)
		}

		// Return JSON response indicating completion
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"isCorrect":   isCorrect,
			"points":      points,
			"correctWord": currentWord.WordText,
			"nextWord":    false,
			"completed":   true,
		})
		return
	}

	// Save timing for next word
	if err := h.practiceService.SaveWordTiming(kid.ID, state.SessionID, newIndex, time.Now()); err != nil {
		log.Printf("Error saving word timing: %v", err)
	}

	// Return JSON response for HTMX
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isCorrect":    isCorrect,
		"points":       points,
		"correctWord":  currentWord.WordText,
		"nextWord":     true,
		"completed":    false,
		"currentIndex": newIndex + 1,
		"totalWords":   len(words),
	})
}

// ShowResults displays practice session results
func (h *PracticeHandler) ShowResults(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
		return
	}

	// Get practice state from database
	state, _, err := h.practiceService.GetPracticeState(kid.ID)
	if err != nil || state == nil {
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// Get session results
	session, attempts, err := h.practiceService.GetSessionResults(state.SessionID)
	if err != nil {
		log.Printf("Error getting session results: %v", err)
		http.Error(w, "Failed to load results", http.StatusInternalServerError)
		return
	}

	// Calculate accuracy
	accuracy := 0.0
	if session.TotalWords > 0 {
		accuracy = float64(session.CorrectWords) / float64(session.TotalWords) * 100
	}

	// Get total points for kid
	totalPoints, err := h.practiceService.GetKidTotalPoints(kid.ID)
	if err != nil {
		log.Printf("Error getting total points: %v", err)
		totalPoints = session.PointsEarned
	}

	data := map[string]interface{}{
		"Title":         "Results - WordClash",
		"Kid":           kid,
		"Session":       session,
		"Attempts":      attempts,
		"Accuracy":      accuracy,
		"TotalPoints":   totalPoints,
	}

	// Clean up practice state from database
	if err := h.practiceService.DeletePracticeState(kid.ID); err != nil {
		log.Printf("Error deleting practice state: %v", err)
	}
	if err := h.practiceService.DeleteWordTimings(kid.ID, state.SessionID); err != nil {
		log.Printf("Error deleting word timings: %v", err)
	}

	if err := h.templates.ExecuteTemplate(w, "results.tmpl", data); err != nil {
		log.Printf("Error rendering results template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
