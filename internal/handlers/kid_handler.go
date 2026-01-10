package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"wordclash/internal/models"
	"wordclash/internal/repository"
	"wordclash/internal/service"
)

// KidHandler handles kid-related HTTP requests
type KidHandler struct {
	familyService   *service.FamilyService
	listService     *service.ListService
	practiceService *service.PracticeService
	middleware      *Middleware
	templates       *template.Template
}

// NewKidHandler creates a new kid handler
func NewKidHandler(familyService *service.FamilyService, listService *service.ListService, practiceService *service.PracticeService, middleware *Middleware, templates *template.Template) *KidHandler {
	return &KidHandler{
		familyService:   familyService,
		listService:     listService,
		practiceService: practiceService,
		middleware:      middleware,
		templates:       templates,
	}
}

// ShowKidSelect displays the kid profile selection page
func (h *KidHandler) ShowKidSelect(w http.ResponseWriter, r *http.Request) {
	// Check if kid is already logged in
	if cookie, err := r.Cookie("kid_session_id"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// Check for error parameter
	hasError := r.URL.Query().Get("error") == "invalid"

	data := map[string]interface{}{
		"Title":    "Select Your Profile - WordClash",
		"HasError": hasError,
	}

	if err := h.templates.ExecuteTemplate(w, "kid_select.tmpl", data); err != nil {
		log.Printf("Error rendering kid select template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// KidLogin handles kid "login" (simple profile selection)
func (h *KidHandler) KidLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Get kid by ID (from old URL format /kid/login/{id})
		kidIDStr := r.PathValue("id")
		if kidIDStr == "" {
			http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
			return
		}
		
		kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid kid ID", http.StatusBadRequest)
			return
		}

		// Get kid to show username
		kid, err := h.familyService.GetKid(kidID)
		if err != nil || kid == nil {
			http.Error(w, "Kid not found", http.StatusNotFound)
			return
		}

		// Check for error parameter
		hasError := r.URL.Query().Get("error") == "invalid"

		data := map[string]interface{}{
			"Title":    "Login - WordClash",
			"Kid":      kid,
			"HasError": hasError,
		}

		if err := h.templates.ExecuteTemplate(w, "kid_login.tmpl", data); err != nil {
			log.Printf("Error rendering kid login template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Handle POST - verify username/password and login
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Get kid by username
	kid, err := h.familyService.GetKidByUsername(username)
	if err != nil {
		log.Printf("Error getting kid by username: %v", err)
		http.Redirect(w, r, "/kid/select?error=invalid", http.StatusSeeOther)
		return
	}
	if kid == nil {
		http.Redirect(w, r, "/kid/select?error=invalid", http.StatusSeeOther)
		return
	}

	// If password is provided, verify it
	if password != "" {
		if kid.Password != password {
			// Redirect back to password page with error
			http.Redirect(w, r, "/kid/login/"+strconv.FormatInt(kid.ID, 10)+"?error=invalid", http.StatusSeeOther)
			return
		}

		// Password correct - create session
		sessionID, expiresAt, err := h.familyService.CreateKidSession(kid.ID)
		if err != nil {
			log.Printf("Error creating kid session: %v", err)
			http.Error(w, "Failed to login", http.StatusInternalServerError)
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "kid_session_id",
			Value:    sessionID,
			Path:     "/",
			Expires:  expiresAt,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// No password provided - redirect to password page
	http.Redirect(w, r, "/kid/login/"+strconv.FormatInt(kid.ID, 10), http.StatusSeeOther)
}

// KidDashboard displays the kid dashboard
func (h *KidHandler) KidDashboard(w http.ResponseWriter, r *http.Request) {
	kid := GetKidFromContext(r.Context())
	if kid == nil {
		http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
		return
	}

	// Get assigned spelling lists
	assignedLists, err := h.listService.GetKidAssignedLists(kid.ID)
	if err != nil {
		log.Printf("Error getting assigned lists: %v", err)
		assignedLists = []models.SpellingList{}
	}

	// Get total points
	totalPoints, err := h.practiceService.GetKidTotalPoints(kid.ID)
	if err != nil {
		log.Printf("Error getting total points: %v", err)
		totalPoints = 0
	}

	// Get recent practice sessions
	recentSessions, err := h.practiceService.GetKidRecentSessions(kid.ID, 5)
	if err != nil {
		log.Printf("Error getting recent sessions: %v", err)
		recentSessions = []models.PracticeSession{}
	}

	data := map[string]interface{}{
		"Title":          "My Dashboard - WordClash",
		"Kid":            kid,
		"AssignedLists":  assignedLists,
		"TotalPoints":    totalPoints,
		"RecentSessions": recentSessions,
	}

	if err := h.templates.ExecuteTemplate(w, "kid_dashboard.tmpl", data); err != nil {
		log.Printf("Error rendering kid dashboard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// KidLogout handles kid logout
func (h *KidHandler) KidLogout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from cookie
	cookie, err := r.Cookie("kid_session_id")
	if err == nil {
		// Delete session from database
		if err := h.familyService.LogoutKid(cookie.Value); err != nil {
			log.Printf("Error logging out kid: %v", err)
		}
	}

	// Clear kid session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "kid_session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
}

// GetKidStrugglingWords returns struggling words data for a kid (for parent view)
func (h *KidHandler) GetKidStrugglingWords(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get kid ID from URL
	kidIDStr := r.PathValue("kidId")
	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	// Get kid to verify access
	kid, err := h.familyService.GetKid(kidID)
	if err != nil {
		log.Printf("Error getting kid: %v", err)
		http.Error(w, "Kid not found", http.StatusNotFound)
		return
	}

	// Verify user has access to this kid's family
	if err := h.familyService.VerifyFamilyAccess(user.ID, kid.FamilyID); err != nil {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get struggling words
	strugglingWords, err := h.practiceService.GetStrugglingWords(kidID)
	if err != nil {
		log.Printf("Error getting struggling words: %v", err)
		http.Error(w, "Failed to get struggling words", http.StatusInternalServerError)
		return
	}

	// Get kid stats
	stats, err := h.practiceService.GetKidStats(kidID)
	if err != nil {
		log.Printf("Error getting kid stats: %v", err)
		http.Error(w, "Failed to get stats", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Kid":             kid,
		"StrugglingWords": strugglingWords,
		"Stats":           stats,
	}

	if err := h.templates.ExecuteTemplate(w, "struggling_words_modal.tmpl", data); err != nil {
		log.Printf("Error rendering struggling words template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// GetKidDetails returns full kid details modal (for parent view)
func (h *KidHandler) GetKidDetails(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get kid ID from URL
	kidIDStr := r.PathValue("id")
	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	// Get kid to verify access
	kid, err := h.familyService.GetKid(kidID)
	if err != nil {
		log.Printf("Error getting kid: %v", err)
		http.Error(w, "Kid not found", http.StatusNotFound)
		return
	}

	// Verify user has access to this kid's family
	if err := h.familyService.VerifyFamilyAccess(user.ID, kid.FamilyID); err != nil {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get assigned lists
	assignedLists, err := h.listService.GetKidAssignedLists(kidID)
	if err != nil {
		log.Printf("Error getting assigned lists: %v", err)
		assignedLists = []models.SpellingList{}
	}

	// Get all available lists for assignment
	allLists, err := h.listService.GetAllUserListsWithAssignments(user.ID)
	if err != nil {
		log.Printf("Error getting all lists: %v", err)
		allLists = []models.ListSummary{}
	}

	// Get struggling words
	strugglingWords, err := h.practiceService.GetStrugglingWords(kidID)
	if err != nil {
		log.Printf("Error getting struggling words: %v", err)
		strugglingWords = []repository.StrugglingWord{}
	}

	// Get kid stats
	stats, err := h.practiceService.GetKidStats(kidID)
	if err != nil {
		log.Printf("Error getting kid stats: %v", err)
		stats = &models.KidStats{}
	}

	// Get CSRF token
	csrfToken := ""
	if cookie, err := r.Cookie("session_id"); err == nil {
		csrfToken, _ = h.middleware.GetCSRFToken(cookie.Value)
	}

	data := map[string]interface{}{
		"Kid":             kid,
		"AssignedLists":   assignedLists,
		"AllLists":        allLists,
		"StrugglingWords": strugglingWords,
		"Stats":           stats,
		"CSRFToken":       csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "kid_detail_modal.tmpl", data); err != nil {
		log.Printf("Error rendering kid detail modal template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

