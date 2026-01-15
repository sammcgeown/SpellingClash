package handlers

import (
	"html/template"
	"log"
	"net/http"
	"spellingclash/internal/models"
	"spellingclash/internal/repository"
	"spellingclash/internal/service"
	"spellingclash/internal/utils"
	"strconv"
)

// AdminHandler handles admin-specific routes
type AdminHandler struct {
	templates    *template.Template
	authService  *service.AuthService
	listService  *service.ListService
	listRepo     *repository.ListRepository
	userRepo     *repository.UserRepository
	familyRepo   *repository.FamilyRepository
	middleware   *Middleware
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(templates *template.Template, authService *service.AuthService, listService *service.ListService, listRepo *repository.ListRepository, userRepo *repository.UserRepository, familyRepo *repository.FamilyRepository, middleware *Middleware) *AdminHandler {
	return &AdminHandler{
		templates:   templates,
		authService: authService,
		listService: listService,
		listRepo:    listRepo,
		userRepo:    userRepo,
		familyRepo:  familyRepo,
		middleware:  middleware,
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

	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":       "Admin Dashboard",
		"User":        user,
		"PublicLists": publicLists,
		"CSRFToken":   csrfToken,
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

// getCSRFToken is a helper to get CSRF token from session
func (h *AdminHandler) getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	token, _ := h.middleware.GetCSRFToken(cookie.Value)
	return token
}

// ShowManageParents shows the parent management page
func (h *AdminHandler) ShowManageParents(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":     "Manage Parents",
		"User":      user,
		"Users":     users,
		"CSRFToken": csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_parents.tmpl", data); err != nil {
		log.Printf("Error rendering admin parents template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// UpdateParent updates a parent's information
func (h *AdminHandler) UpdateParent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	name := r.FormValue("name")
	isAdmin := r.FormValue("is_admin") == "on"

	// Update user info
	if err := h.userRepo.UpdateUser(userID, email, name, isAdmin); err != nil {
		log.Printf("Error updating user: %v", err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// CreateParent creates a new parent user
func (h *AdminHandler) CreateParent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	isAdminStr := r.FormValue("is_admin")

	if name == "" || email == "" || password == "" {
		http.Error(w, "Name, email, and password are required", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Create the user (note: CreateUser expects email, passwordHash, name)
	newUser, err := h.userRepo.CreateUser(email, hashedPassword, name)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Update admin status if requested
	if isAdminStr == "on" || isAdminStr == "true" {
		if err := h.userRepo.UpdateUser(newUser.ID, newUser.Name, newUser.Email, true); err != nil {
			log.Printf("Error setting admin status: %v", err)
		}
	}

	// Auto-create a family for the new user
	familyName := name + "'s Family"
	if _, err := h.familyRepo.CreateFamily(familyName, newUser.ID); err != nil {
		log.Printf("Error creating family for new user: %v", err)
		// Don't fail the whole operation if family creation fails
	}

	http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// DeleteParent deletes a parent user
func (h *AdminHandler) DeleteParent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent deleting yourself
	if userID == user.ID {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	if err := h.userRepo.DeleteUser(userID); err != nil {
		log.Printf("Error deleting user: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// ShowManageFamilies shows the family management page
func (h *AdminHandler) ShowManageFamilies(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	families, err := h.familyRepo.GetAllFamilies()
	if err != nil {
		log.Printf("Error fetching families: %v", err)
		http.Error(w, "Failed to load families", http.StatusInternalServerError)
		return
	}

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	// Get members for each family
	familyMembers := make(map[int64][]models.User)
	for _, family := range families {
		_, members, err := h.familyRepo.GetFamilyMembers(family.ID)
		if err != nil {
			log.Printf("Error fetching members for family %d: %v", family.ID, err)
			continue
		}
		familyMembers[family.ID] = members
	}

	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":         "Manage Families",
		"User":          user,
		"Families":      families,
		"Users":         users,
		"FamilyMembers": familyMembers,
		"CSRFToken":     csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_families.tmpl", data); err != nil {
		log.Printf("Error rendering admin families template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// CreateFamily creates a new family
func (h *AdminHandler) CreateFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Family name is required", http.StatusBadRequest)
		return
	}

	_, err := h.familyRepo.CreateFamily(name, user.ID)
	if err != nil {
		log.Printf("Error creating family: %v", err)
		http.Error(w, "Failed to create family", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/families", http.StatusSeeOther)
}

// UpdateFamily updates a family's information
func (h *AdminHandler) UpdateFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	familyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid family ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	memberIDs := r.Form["member_ids"]

	// Update family name
	if err := h.familyRepo.UpdateFamily(familyID, name); err != nil {
		log.Printf("Error updating family: %v", err)
		http.Error(w, "Failed to update family", http.StatusInternalServerError)
		return
	}

	// Get current members
	_, currentMembers, err := h.familyRepo.GetFamilyMembers(familyID)
	if err != nil {
		log.Printf("Error fetching current members: %v", err)
		http.Error(w, "Failed to fetch current members", http.StatusInternalServerError)
		return
	}

	// Build map of current member IDs
	currentMemberMap := make(map[int64]bool)
	for _, m := range currentMembers {
		currentMemberMap[m.ID] = true
	}

	// Build map of new member IDs
	newMemberMap := make(map[int64]bool)
	for _, midStr := range memberIDs {
		mid, err := strconv.ParseInt(midStr, 10, 64)
		if err != nil {
			continue
		}
		newMemberMap[mid] = true
	}

	// Remove members that are no longer selected
	for _, m := range currentMembers {
		if !newMemberMap[m.ID] {
			if err := h.familyRepo.RemoveUserFromFamily(m.ID, familyID); err != nil {
				log.Printf("Error removing user from family: %v", err)
			}
		}
	}

	// Add new members
	for midStr := range newMemberMap {
		if !currentMemberMap[midStr] {
			if err := h.familyRepo.AddUserToFamily(midStr, familyID); err != nil {
				log.Printf("Error adding user to family: %v", err)
			}
		}
	}

	http.Redirect(w, r, "/admin/families", http.StatusSeeOther)
}

// DeleteFamily deletes a family
func (h *AdminHandler) DeleteFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	familyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid family ID", http.StatusBadRequest)
		return
	}

	if err := h.familyRepo.DeleteFamily(familyID); err != nil {
		log.Printf("Error deleting family: %v", err)
		http.Error(w, "Failed to delete family", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/families", http.StatusSeeOther)
}
