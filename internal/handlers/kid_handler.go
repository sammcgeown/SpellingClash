package handlers

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
	"wordclash/internal/models"
	"wordclash/internal/service"
)

// KidHandler handles kid-related HTTP requests
type KidHandler struct {
	familyService *service.FamilyService
	templates     *template.Template
}

// NewKidHandler creates a new kid handler
func NewKidHandler(familyService *service.FamilyService, templates *template.Template) *KidHandler {
	return &KidHandler{
		familyService: familyService,
		templates:     templates,
	}
}

// ShowKidSelect displays the kid profile selection page
func (h *KidHandler) ShowKidSelect(w http.ResponseWriter, r *http.Request) {
	// Check if kid is already logged in
	if cookie, err := r.Cookie("kid_session_id"); err == nil && cookie.Value != "" {
		http.Redirect(w, r, "/kid/dashboard", http.StatusSeeOther)
		return
	}

	// For now, we'll get all kids from the database
	// In a real app, you might want to scope this to a specific parent or family
	// For simplicity, we'll show all kids
	data := map[string]interface{}{
		"Title": "Select Your Profile - WordClash",
		"Kids":  []models.Kid{}, // Will be populated by parent dashboard
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

	// Verify kid exists
	kid, err := h.familyService.GetKid(kidID)
	if err != nil {
		log.Printf("Error getting kid: %v", err)
		http.Error(w, "Kid not found", http.StatusNotFound)
		return
	}

	// Create a simple session cookie with kid ID
	// In a production app, you might want to create a proper session in the database
	http.SetCookie(w, &http.Cookie{
		Name:     "kid_session_id",
		Value:    strconv.FormatInt(kid.ID, 10),
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
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

	data := map[string]interface{}{
		"Title": "My Dashboard - WordClash",
		"Kid":   kid,
	}

	if err := h.templates.ExecuteTemplate(w, "kid_dashboard.tmpl", data); err != nil {
		log.Printf("Error rendering kid dashboard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// KidLogout handles kid logout
func (h *KidHandler) KidLogout(w http.ResponseWriter, r *http.Request) {
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

// GetKidFromContext retrieves the kid from the request context
func GetKidFromContext(ctx context.Context) *models.Kid {
	kid, ok := ctx.Value(KidSessionContextKey).(*models.Kid)
	if !ok {
		return nil
	}
	return kid
}
