package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"spellingclash/internal/models"
	"spellingclash/internal/repository"
	"spellingclash/internal/security"
	"spellingclash/internal/service"
)

// AdminHandler handles admin-specific routes
type AdminHandler struct {
	templates     *template.Template
	authService   *service.AuthService
	listService   *service.ListService
	backupService *service.BackupService
	listRepo      *repository.ListRepository
	userRepo      *repository.UserRepository
	familyRepo    *repository.FamilyRepository
	kidRepo       *repository.KidRepository
	middleware    *Middleware
	version       string
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(templates *template.Template, authService *service.AuthService, listService *service.ListService, backupService *service.BackupService, listRepo *repository.ListRepository, userRepo *repository.UserRepository, familyRepo *repository.FamilyRepository, kidRepo *repository.KidRepository, middleware *Middleware, version string) *AdminHandler {
	return &AdminHandler{
		templates:     templates,
		authService:   authService,
		listService:   listService,
		backupService: backupService,
		listRepo:      listRepo,
		userRepo:      userRepo,
		familyRepo:    familyRepo,
		kidRepo:       kidRepo,
		middleware:    middleware,
		version:       version,
	}
}

// ShowAdminDashboard shows the admin dashboard
func (h *AdminHandler) ShowAdminDashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	publicLists, err := h.listRepo.GetPublicLists()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load public lists", "Error fetching public lists", err)
		return
	}

	csrfToken := h.getCSRFToken(r)

	data := AdminDashboardViewData{
		Title:       "Admin Dashboard",
		User:        user,
		PublicLists: publicLists,
		CSRFToken:   csrfToken,
		Version:     h.version,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_dashboard.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerErrorUC, "Error rendering admin dashboard", err)
	}
}

// RegeneratePublicLists regenerates all public lists from the data files
func (h *AdminHandler) RegeneratePublicLists(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Delete existing public lists
	publicLists, err := h.listRepo.GetPublicLists()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch public lists", "Error fetching public lists", err)
		return
	}

	for _, list := range publicLists {
		if err := h.listRepo.DeleteList(list.ID); err != nil {
			log.Printf("Error deleting list %d: %v", list.ID, err)
		}
	}

	// Regenerate public lists
	if err := h.listService.SeedDefaultPublicLists(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to regenerate public lists", "Error seeding public lists", err)
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
	cookie, err := r.Cookie(SessionCookieName)
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
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load users", "Error fetching users", err)
		return
	}

	// Create a slice with user and family code combined
	usersWithFamily := make([]AdminUserWithFamily, 0, len(users))
	for _, u := range users {
		uwf := AdminUserWithFamily{User: u}
		families, err := h.familyRepo.GetUserFamilies(u.ID)
		if err != nil {
			log.Printf("Error fetching families for user %d: %v", u.ID, err)
		} else if len(families) > 0 {
			uwf.FamilyCode = families[0].FamilyCode
		}
		usersWithFamily = append(usersWithFamily, uwf)
	}

	csrfToken := h.getCSRFToken(r)

	data := AdminParentsViewData{
		Title:     "Manage Parents",
		User:      user,
		Users:     usersWithFamily,
		CSRFToken: csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_parents.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerErrorUC, "Error rendering admin parents template", err)
	}
}

// UpdateParent updates a parent's information
func (h *AdminHandler) UpdateParent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
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
		respondWithError(w, http.StatusInternalServerError, "Failed to update user", "Error updating user", err)
		return
	}

	http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// CreateParent creates a new parent user
func (h *AdminHandler) CreateParent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
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
	hashedPassword, err := security.HashPassword(password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user", "Error hashing password", err)
		return
	}

	// Create the user (note: CreateUser expects email, passwordHash, name)
	newUser, err := h.userRepo.CreateUser(email, hashedPassword, name)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create user", "Error creating user", err)
		return
	}

	// Update admin status if requested
	if isAdminStr == "on" || isAdminStr == "true" {
		if err := h.userRepo.UpdateUser(newUser.ID, newUser.Name, newUser.Email, true); err != nil {
			log.Printf("Error setting admin status: %v", err)
		}
	}

	// Auto-create a family for the new user
	if _, err := h.familyRepo.CreateFamily(newUser.ID); err != nil {
		log.Printf("Error creating family for new user: %v", err)
		// Don't fail the whole operation if family creation fails
	}

	http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// DeleteParent deletes a parent user
func (h *AdminHandler) DeleteParent(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
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
		respondWithError(w, http.StatusInternalServerError, "Failed to delete user", "Error deleting user", err)
		return
	}

	http.Redirect(w, r, "/admin/parents", http.StatusSeeOther)
}

// ShowManageFamilies shows the family management page
func (h *AdminHandler) ShowManageFamilies(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	families, err := h.familyRepo.GetAllFamilies()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load families", "Error fetching families", err)
		return
	}

	users, err := h.userRepo.GetAllUsers()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load users", "Error fetching users", err)
		return
	}

	// Get members for each family
	familyMembers := make(map[string][]models.User)
	for _, family := range families {
		_, members, err := h.familyRepo.GetFamilyMembers(family.FamilyCode)
		if err != nil {
			log.Printf("Error fetching members for family %s: %v", family.FamilyCode, err)
			continue
		}
		familyMembers[family.FamilyCode] = members
	}

	csrfToken := h.getCSRFToken(r)

	data := AdminFamiliesViewData{
		Title:         "Manage Families",
		User:          user,
		Families:      families,
		Users:         users,
		FamilyMembers: familyMembers,
		CSRFToken:     csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_families.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerErrorUC, "Error rendering admin families template", err)
	}
}

// CreateFamily creates a new family
func (h *AdminHandler) CreateFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	userIDStr := r.FormValue("user_id")
	if userIDStr == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	_, err = h.familyRepo.CreateFamily(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create family", "Error creating family", err)
		return
	}

	http.Redirect(w, r, "/admin/families", http.StatusSeeOther)
}

// ExportDatabase exports the database to JSON for download
func (h *AdminHandler) ExportDatabase(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Set headers for file download
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("spellingclash_backup_%s.json", timestamp)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Export directly to response writer
	if err := h.backupService.ExportToWriter(w); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to export database", "Error exporting database", err)
		return
	}

	log.Printf("Database exported by admin user %s", user.Email)
}

// ShowDatabaseManagement shows the database backup/restore page
func (h *AdminHandler) ShowDatabaseManagement(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Get database statistics
	stats, err := h.getDatabaseStats()
	if err != nil {
		log.Printf("Error getting database stats: %v", err)
		stats = &DatabaseStats{}
	}

	data := AdminDatabaseViewData{
		Title:     "Database Management - SpellingClash Admin",
		User:      user,
		Stats:     stats,
		CSRFToken: h.getCSRFToken(r),
	}

	if err := h.templates.ExecuteTemplate(w, "admin_database.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerError, "Error rendering database template", err)
	}
}

// ImportDatabase imports a database backup from uploaded file
func (h *AdminHandler) ImportDatabase(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("backup_file")
	if err != nil {
		h.showDatabasePageWithError(w, r, "Please select a backup file")
		return
	}
	defer file.Close()

	clearData := r.FormValue("clear_data") == "true"

	// Clear database if requested
	if clearData {
		log.Printf("Admin %s requested database clear before import", user.Email)
		if err := h.clearDatabase(); err != nil {
			log.Printf("Error clearing database: %v", err)
			h.showDatabasePageWithError(w, r, "Failed to clear database: "+err.Error())
			return
		}
	}

	// Import from reader
	if err := h.backupService.ImportFromReader(file); err != nil {
		log.Printf("Error importing database: %v", err)
		h.showDatabasePageWithError(w, r, "Failed to import database: "+err.Error())
		return
	}

	log.Printf("Database imported successfully by admin user %s (clear_data=%v)", user.Email, clearData)
	h.showDatabasePageWithSuccess(w, r, "Database imported successfully!")
}

// DatabaseStats holds database statistics
type DatabaseStats struct {
	Users     int
	Families  int
	Kids      int
	Lists     int
	Words     int
	Practices int
}

func (h *AdminHandler) getDatabaseStats() (*DatabaseStats, error) {
	stats := &DatabaseStats{}
	db := h.backupService.GetDB()

	// Count users
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.Users); err != nil {
		return nil, err
	}

	// Count families
	if err := db.QueryRow("SELECT COUNT(*) FROM families").Scan(&stats.Families); err != nil {
		return nil, err
	}

	// Count kids
	if err := db.QueryRow("SELECT COUNT(*) FROM kids").Scan(&stats.Kids); err != nil {
		return nil, err
	}

	// Count lists
	if err := db.QueryRow("SELECT COUNT(*) FROM spelling_lists").Scan(&stats.Lists); err != nil {
		return nil, err
	}

	// Count words
	if err := db.QueryRow("SELECT COUNT(*) FROM words").Scan(&stats.Words); err != nil {
		return nil, err
	}

	// Count practice sessions
	if err := db.QueryRow("SELECT COUNT(*) FROM practice_sessions").Scan(&stats.Practices); err != nil {
		return nil, err
	}

	return stats, nil
}

func (h *AdminHandler) clearDatabase() error {
	db := h.backupService.GetDB()

	// Delete in reverse order of dependencies
	tables := []string{
		"practice_results", // May not exist in current schema, but try to clear it anyway
		"practice_sessions",
		"list_assignments",
		"words",
		"spelling_lists",
		"kid_sessions",
		"kids",
		"family_members",
		"families",
		"password_reset_tokens",
		"sessions",
		"users",
	}

	allowedTables := map[string]struct{}{
		"practice_results":      {},
		"practice_sessions":     {},
		"list_assignments":      {},
		"words":                 {},
		"spelling_lists":        {},
		"kid_sessions":          {},
		"kids":                  {},
		"family_members":        {},
		"families":              {},
		"password_reset_tokens": {},
		"sessions":              {},
		"users":                 {},
	}

	for _, table := range tables {
		if _, ok := allowedTables[table]; !ok {
			return fmt.Errorf("invalid table name: %s", table)
		}
		query := fmt.Sprintf("DELETE FROM %s", table)
		if _, err := db.Exec(query); err != nil {
			// Ignore "no such table" errors for backwards compatibility
			if !strings.Contains(err.Error(), "no such table") &&
				!strings.Contains(err.Error(), "doesn't exist") &&
				!strings.Contains(err.Error(), "does not exist") {
				return fmt.Errorf("failed to clear table %s: %w", table, err)
			}
		}
	}

	return nil
}

func (h *AdminHandler) showDatabasePageWithError(w http.ResponseWriter, r *http.Request, errMsg string) {
	user := GetUserFromContext(r.Context())
	stats, _ := h.getDatabaseStats()

	data := AdminDatabaseViewData{
		Title:     "Database Management - SpellingClash Admin",
		User:      user,
		Stats:     stats,
		CSRFToken: h.getCSRFToken(r),
		Error:     errMsg,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_database.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerError, "Error rendering database template", err)
	}
}

func (h *AdminHandler) showDatabasePageWithSuccess(w http.ResponseWriter, r *http.Request, msg string) {
	user := GetUserFromContext(r.Context())
	stats, _ := h.getDatabaseStats()

	data := AdminDatabaseViewData{
		Title:     "Database Management - SpellingClash Admin",
		User:      user,
		Stats:     stats,
		CSRFToken: h.getCSRFToken(r),
		Success:   msg,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_database.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerError, "Error rendering database template", err)
	}
}

// UpdateFamily updates a family's member list
func (h *AdminHandler) UpdateFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	familyCode := r.PathValue("code")
	if familyCode == "" {
		http.Error(w, "Invalid family code", http.StatusBadRequest)
		return
	}

	memberIDs := r.Form["member_ids"]

	// Get current members
	_, currentMembers, err := h.familyRepo.GetFamilyMembers(familyCode)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch current members", "Error fetching current members", err)
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
			if err := h.familyRepo.RemoveUserFromFamily(m.ID, familyCode); err != nil {
				log.Printf("Error removing user from family: %v", err)
			}
		}
	}

	// Add new members
	for midInt := range newMemberMap {
		if !currentMemberMap[midInt] {
			if err := h.familyRepo.AddUserToFamily(midInt, familyCode); err != nil {
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
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	familyCode := r.PathValue("code")
	if familyCode == "" {
		http.Error(w, "Invalid family code", http.StatusBadRequest)
		return
	}

	if err := h.familyRepo.DeleteFamily(familyCode); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete family", "Error deleting family", err)
		return
	}

	http.Redirect(w, r, "/admin/families", http.StatusSeeOther)
}

// ShowManageKids shows the kids management page
func (h *AdminHandler) ShowManageKids(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	kids, err := h.kidRepo.GetAllKids()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to load kids", "Error fetching kids", err)
		return
	}

	csrfToken := h.getCSRFToken(r)

	data := AdminKidsViewData{
		Title:     "Manage Kids",
		User:      user,
		Kids:      kids,
		CSRFToken: csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "admin_kids.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerErrorUC, "Error rendering admin kids template", err)
	}
}

// UpdateKid updates a kid's information
func (h *AdminHandler) UpdateKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	kidID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	avatarColor := r.FormValue("avatar_color")
	password := r.FormValue("password")

	// Update basic kid info
	if err := h.kidRepo.UpdateKid(kidID, name, avatarColor); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update kid", "Error updating kid", err)
		return
	}

	// Update password if provided
	if password != "" {
		if err := h.kidRepo.UpdateKidPassword(kidID, password); err != nil {
			log.Printf("Error updating kid password: %v", err)
		}
	}

	// Note: Username updates would require a separate UpdateKidUsername method in kid_repo.go
	// For now, username is not updated via this admin interface

	http.Redirect(w, r, "/admin/kids", http.StatusSeeOther)
}

// DeleteKid deletes a kid
func (h *AdminHandler) DeleteKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil || !user.IsAdmin {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}

	kidID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	if err := h.kidRepo.DeleteKid(kidID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete kid", "Error deleting kid", err)
		return
	}

	http.Redirect(w, r, "/admin/kids", http.StatusSeeOther)
}
