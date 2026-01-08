package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
	"wordclash/internal/models"
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

// PracticeState holds the current state of a practice session in memory
type PracticeState struct {
	SessionID      int64
	Words          []models.Word
	CurrentIndex   int
	CorrectCount   int
	TotalPoints    int
	StartTime      time.Time
	WordStartTimes map[int]time.Time // Track when each word was presented
}

// In-memory storage for practice states (in production, use Redis or similar)
var practiceStates = make(map[int64]*PracticeState) // kidID -> PracticeState

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

	// Store practice state
	practiceStates[kid.ID] = &PracticeState{
		SessionID:      session.ID,
		Words:          words,
		CurrentIndex:   0,
		CorrectCount:   0,
		TotalPoints:    0,
		StartTime:      time.Now(),
		WordStartTimes: make(map[int]time.Time),
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

	// Get practice state
	state, exists := practiceStates[kid.ID]
	if !exists || state.CurrentIndex >= len(state.Words) {
		// No active session or completed
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// Record word start time
	if _, exists := state.WordStartTimes[state.CurrentIndex]; !exists {
		state.WordStartTimes[state.CurrentIndex] = time.Now()
	}

	currentWord := state.Words[state.CurrentIndex]

	data := map[string]interface{}{
		"Title":        "Practice - WordClash",
		"Kid":          kid,
		"Word":         currentWord,
		"CurrentIndex": state.CurrentIndex + 1,
		"TotalWords":   len(state.Words),
		"CorrectCount": state.CorrectCount,
		"TotalPoints":  state.TotalPoints,
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

	// Get practice state
	state, exists := practiceStates[kid.ID]
	if !exists {
		http.Error(w, "No active practice session", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	answer := r.FormValue("answer")
	currentWord := state.Words[state.CurrentIndex]

	// Calculate time taken
	startTime := state.WordStartTimes[state.CurrentIndex]
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
	if isCorrect {
		state.CorrectCount++
	}
	state.TotalPoints += points
	state.CurrentIndex++

	// Check if session is complete
	if state.CurrentIndex >= len(state.Words) {
		// Complete the session
		_, err := h.practiceService.CompleteSession(state.SessionID)
		if err != nil {
			log.Printf("Error completing session: %v", err)
		}

		// Redirect to results
		http.Redirect(w, r, "/kid/practice/results", http.StatusSeeOther)
		return
	}

	// Return JSON response for HTMX
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"isCorrect":    isCorrect,
		"points":       points,
		"correctWord":  currentWord.WordText,
		"nextWord":     true,
		"currentIndex": state.CurrentIndex + 1,
		"totalWords":   len(state.Words),
	})
}

// ShowResults displays practice session results
func (h *PracticeHandler) ShowResults(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
		return
	}

	// Get practice state
	state, exists := practiceStates[kid.ID]
	if !exists {
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

	// Clean up practice state
	delete(practiceStates, kid.ID)

	if err := h.templates.ExecuteTemplate(w, "results.tmpl", data); err != nil {
		log.Printf("Error rendering results template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
