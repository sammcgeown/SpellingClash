package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"wordclash/internal/models"
	"wordclash/internal/service"
)

// KidHandler handles kid-related HTTP requests
type KidHandler struct {
	familyService   *service.FamilyService
	listService     *service.ListService
	practiceService *service.PracticeService
	templates       *template.Template
}

// NewKidHandler creates a new kid handler
func NewKidHandler(familyService *service.FamilyService, listService *service.ListService, practiceService *service.PracticeService, templates *template.Template) *KidHandler {
	return &KidHandler{
		familyService:   familyService,
		listService:     listService,
		practiceService: practiceService,
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

	// Get all kids from all families
	// In a production app, you might want to scope this to specific families
	kids, err := h.familyService.GetAllKids()
	if err != nil {
		log.Printf("Error getting kids: %v", err)
		kids = []models.Kid{}
	}

	data := map[string]interface{}{
		"Title": "Select Your Profile - WordClash",
		"Kids":  kids,
	}

	if err := h.templates.ExecuteTemplate(w, "kid_select.tmpl", data); err != nil {
		log.Printf("Error rendering kid select template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// KidLogin handles kid "login" (simple profile selection)
func (h *KidHandler) KidLogin(w http.ResponseWriter, r *http.Request) {
	kidIDStr := r.PathValue("id")
	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	// Create a proper kid session
	sessionID, expiresAt, err := h.familyService.CreateKidSession(kidID)
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
