package handlers

import (
	"log"
	"net/http"
	"spellingclash/internal/repository"
	"spellingclash/internal/service"
	"html/template"
)

// AdminHandler handles admin-specific routes
type AdminHandler struct {
	templates   *template.Template
	authService *service.AuthService
	listService *service.ListService
	listRepo    *repository.ListRepository
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(templates *template.Template, authService *service.AuthService, listService *service.ListService, listRepo *repository.ListRepository) *AdminHandler {
	return &AdminHandler{
		templates:   templates,
		authService: authService,
		listService: listService,
		listRepo:    listRepo,
	}
}

// ShowAdminDashboard shows the admin dashboard
func (h *AdminHandler) ShowAdminDashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	publicLists, err := h.listRepo.GetPublicLists()
	if err != nil {
		log.Printf("Error fetching public lists: %v", err)
		http.Error(w, "Failed to load public lists", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":       "Admin Dashboard",
		"User":        user,
		"PublicLists": publicLists,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_dashboard.tmpl", data); err != nil {
		log.Printf("Error rendering admin dashboard: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// RegeneratePublicLists regenerates all public lists from the data files
func (h *AdminHandler) RegeneratePublicLists(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Delete existing public lists
	publicLists, err := h.listRepo.GetPublicLists()
	if err != nil {
		log.Printf("Error fetching public lists: %v", err)
		http.Error(w, "Failed to fetch public lists", http.StatusInternalServerError)
		return
	}

	for _, list := range publicLists {
		if err := h.listRepo.DeleteList(list.ID); err != nil {
			log.Printf("Error deleting list %d: %v", list.ID, err)
		}
	}

	// Regenerate public lists
	if err := h.listService.SeedDefaultPublicLists(); err != nil {
		log.Printf("Error seeding public lists: %v", err)
		http.Error(w, "Failed to regenerate public lists", http.StatusInternalServerError)
		return
	}

	// Regenerate audio files
	if err := h.listService.GenerateMissingAudio(); err != nil {
		log.Printf("Warning: Failed to generate audio files: %v", err)
	}

	http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
}
