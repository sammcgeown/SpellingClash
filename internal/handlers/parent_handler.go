package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"wordclash/internal/models"
	"wordclash/internal/service"
)

// ParentHandler handles parent-related HTTP requests
type ParentHandler struct {
	familyService *service.FamilyService
	listService   *service.ListService
	middleware    *Middleware
	templates     *template.Template
}

// NewParentHandler creates a new parent handler
func NewParentHandler(familyService *service.FamilyService, listService *service.ListService, middleware *Middleware, templates *template.Template) *ParentHandler {
	return &ParentHandler{
		familyService: familyService,
		listService:   listService,
		middleware:    middleware,
		templates:     templates,
	}
}

// Dashboard renders the parent dashboard
func (h *ParentHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get user's families
	families, err := h.familyService.GetUserFamilies(user.ID)
	if err != nil {
		log.Printf("Error getting user families: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get all kids from all families
	allKids, err := h.familyService.GetAllUserKids(user.ID)
	if err != nil {
		log.Printf("Error getting user kids: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get CSRF token
	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":     "Dashboard - WordClash",
		"User":      user,
		"Families":  families,
		"Kids":      allKids,
		"CSRFToken": csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "dashboard.tmpl", data); err != nil {
		log.Printf("Error rendering dashboard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ShowFamily displays family management page
func (h *ParentHandler) ShowFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	families, err := h.familyService.GetUserFamilies(user.ID)
	if err != nil {
		log.Printf("Error getting user families: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get CSRF token
	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":     "Manage Families - WordClash",
		"User":      user,
		"Families":  families,
		"CSRFToken": csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "family.tmpl", data); err != nil {
		log.Printf("Error rendering family template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CreateFamily handles family creation
func (h *ParentHandler) CreateFamily(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")

	_, err := h.familyService.CreateFamily(name, user.ID)
	if err != nil {
		log.Printf("Error creating family: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/family", http.StatusSeeOther)
}

// ShowKids displays kids management page
func (h *ParentHandler) ShowKids(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	families, err := h.familyService.GetUserFamilies(user.ID)
	if err != nil {
		log.Printf("Error getting user families: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	allKids, err := h.familyService.GetAllUserKids(user.ID)
	if err != nil {
		log.Printf("Error getting user kids: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Get assigned lists for each kid
	var kidsWithLists []models.KidWithLists
	for _, kid := range allKids {
		assignedLists, err := h.listService.GetKidAssignedLists(kid.ID)
		if err != nil {
			log.Printf("Error getting assigned lists for kid %d: %v", kid.ID, err)
			assignedLists = []models.SpellingList{}
		}
		kidsWithLists = append(kidsWithLists, models.KidWithLists{
			Kid:           kid,
			AssignedLists: assignedLists,
		})
	}

	// Get all available lists (user's lists + public lists) for assignment
	allLists, err := h.listService.GetAllUserListsWithAssignments(user.ID)
	if err != nil {
		log.Printf("Error getting all lists: %v", err)
		allLists = []models.ListSummary{}
	}

	// Get CSRF token
	csrfToken := h.getCSRFToken(r)

	data := map[string]interface{}{
		"Title":     "Manage Kids - WordClash",
		"User":      user,
		"Families":  families,
		"Kids":      kidsWithLists,
		"AllLists":  allLists,
		"CSRFToken": csrfToken,
	}

	if err := h.templates.ExecuteTemplate(w, "kids.tmpl", data); err != nil {
		log.Printf("Error rendering kids template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// CreateKid handles kid creation
func (h *ParentHandler) CreateKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	familyIDStr := r.FormValue("family_id")
	avatarColor := r.FormValue("avatar_color")

	familyID, err := strconv.ParseInt(familyIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid family ID", http.StatusBadRequest)
		return
	}

	if avatarColor == "" {
		avatarColor = "#4A90E2"
	}

	kid, err := h.familyService.CreateKid(familyID, user.ID, name, avatarColor)
	if err != nil {
		log.Printf("Error creating kid: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If it's an HTMX request, return kid credentials HTML and trigger page reload
	if r.Header.Get("HX-Request") == "true" {
		html := `<div class="credentials-display">
			<h3>✅ Kid Created Successfully!</h3>
			<div class="credentials-box">
				<p><strong>Name:</strong> ` + kid.Name + `</p>
				<p><strong>Username:</strong> <code>` + kid.Username + `</code></p>
				<p><strong>Password:</strong> <code>` + kid.Password + `</code></p>
				<p class="text-muted">⚠️ Please save these credentials! The child will need them to log in.</p>
				<p class="text-muted" style="margin-top: 15px;">This page will refresh in 3 seconds...</p>
			</div>
			<script>
				setTimeout(function() {
					window.location.reload();
				}, 3000);
			</script>
		</div>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}

	http.Redirect(w, r, "/parent/kids", http.StatusSeeOther)
}

// UpdateKid handles kid updates
func (h *ParentHandler) UpdateKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	kidIDStr := r.PathValue("id")
	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	avatarColor := r.FormValue("avatar_color")

	if err := h.familyService.UpdateKid(kidID, user.ID, name, avatarColor); err != nil {
		log.Printf("Error updating kid: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/kids", http.StatusSeeOther)
}

// RegenerateKidPassword generates a new random password for a kid
func (h *ParentHandler) RegenerateKidPassword(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	kidIDStr := r.PathValue("id")
	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	newPassword, err := h.familyService.RegenerateKidPassword(kidID, user.ID)
	if err != nil {
		log.Printf("Error regenerating kid password: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return the new password as plain text for HTMX to update the UI
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(newPassword))
}

// DeleteKid handles kid deletion
func (h *ParentHandler) DeleteKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	kidIDStr := r.PathValue("id")
	kidID, err := strconv.ParseInt(kidIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid kid ID", http.StatusBadRequest)
		return
	}

	if err := h.familyService.DeleteKid(kidID, user.ID); err != nil {
		log.Printf("Error deleting kid: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/parent/kids", http.StatusSeeOther)
}

// getCSRFToken is a helper to get CSRF token from session
func (h *ParentHandler) getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return ""
	}
	token, _ := h.middleware.GetCSRFToken(cookie.Value)
	return token
}
