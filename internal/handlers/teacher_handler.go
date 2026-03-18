package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"spellingclash/internal/models"
	"spellingclash/internal/service"
	"strconv"
	"strings"
)

// TeacherHandler handles teacher-facing class management routes.
type TeacherHandler struct {
	teacherService *service.TeacherService
	middleware     *Middleware
	templates      *template.Template
}

// NewTeacherHandler creates a new teacher handler.
func NewTeacherHandler(teacherService *service.TeacherService, middleware *Middleware, templates *template.Template) *TeacherHandler {
	return &TeacherHandler{
		teacherService: teacherService,
		middleware:     middleware,
		templates:      templates,
	}
}

// Dashboard renders the teacher dashboard.
func (h *TeacherHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !user.IsTeacher {
		http.Error(w, "Forbidden: Teacher access required", http.StatusForbidden)
		return
	}

	kids, err := h.teacherService.GetTeacherKids(user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerError, "Error getting class children", err)
		return
	}

	data := TeacherDashboardViewData{
		Title:     "Teacher Dashboard - WordClash",
		User:      user,
		Kids:      kids,
		CSRFToken: h.getCSRFToken(r),
	}
	if err := h.templates.ExecuteTemplate(w, "teacher_dashboard.tmpl", data); err != nil {
		respondWithError(w, http.StatusInternalServerError, ErrInternalServerError, "Error rendering teacher dashboard", err)
	}
}

// CreateKid creates one child for the teacher.
func (h *TeacherHandler) CreateKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	if !user.IsTeacher {
		http.Error(w, "Forbidden: Teacher access required", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	avatarColor := strings.TrimSpace(r.FormValue("avatar_color"))

	kid, err := h.teacherService.CreateTeacherKid(user.ID, name, avatarColor)
	if err != nil {
		log.Printf("Error creating teacher child account: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fmt.Sprintf(`<div class="credentials-display">
			<h3>Child Account Created</h3>
			<div class="credentials-box">
				<p><strong>Name:</strong> %s</p>
				<p><strong>Username:</strong> <code>%s</code></p>
				<p><strong>Password:</strong> <code>%s</code></p>
			</div>
		</div>`, template.HTMLEscapeString(kid.Name), template.HTMLEscapeString(kid.Username), template.HTMLEscapeString(kid.Password))))
		return
	}

	http.Redirect(w, r, "/teacher/dashboard", http.StatusSeeOther)
}

// BulkCreateKids creates multiple child accounts from newline-separated names.
func (h *TeacherHandler) BulkCreateKids(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	if !user.IsTeacher {
		http.Error(w, "Forbidden: Teacher access required", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	rawNames := r.FormValue("child_names")
	avatarColor := strings.TrimSpace(r.FormValue("avatar_color"))
	names := parseBulkNames(rawNames)
	if len(names) == 0 {
		http.Error(w, "Please provide at least one child name", http.StatusBadRequest)
		return
	}

	kids, err := h.teacherService.BulkCreateTeacherKids(user.ID, names, avatarColor)
	if err != nil {
		log.Printf("Error bulk creating child accounts: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(h.renderBulkCredentials(kids)))
		return
	}

	http.Redirect(w, r, "/teacher/dashboard", http.StatusSeeOther)
}

// LinkExistingKid links a teacher to an existing child account by username.
func (h *TeacherHandler) LinkExistingKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	if !user.IsTeacher {
		http.Error(w, "Forbidden: Teacher access required", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	kid, err := h.teacherService.LinkExistingKidByUsername(user.ID, username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<div class="success-message">Linked child ` + template.HTMLEscapeString(kid.Name) + ` (` + template.HTMLEscapeString(kid.Username) + `) to your class.</div>`))
		return
	}

	http.Redirect(w, r, "/teacher/dashboard", http.StatusSeeOther)
}

// UpdateKid updates a child linked to the teacher.
func (h *TeacherHandler) UpdateKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	if !user.IsTeacher {
		http.Error(w, "Forbidden: Teacher access required", http.StatusForbidden)
		return
	}

	kidID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid child ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, ErrInvalidFormData, http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	avatarColor := strings.TrimSpace(r.FormValue("avatar_color"))
	if err := h.teacherService.UpdateTeacherKid(user.ID, kidID, name, avatarColor); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/teacher/dashboard", http.StatusSeeOther)
}

// DeleteKid deletes a child linked to the teacher.
func (h *TeacherHandler) DeleteKid(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		http.Error(w, ErrUnauthorized, http.StatusUnauthorized)
		return
	}
	if !user.IsTeacher {
		http.Error(w, "Forbidden: Teacher access required", http.StatusForbidden)
		return
	}

	kidID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid child ID", http.StatusBadRequest)
		return
	}

	if err := h.teacherService.DeleteTeacherKid(user.ID, kidID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/teacher/dashboard", http.StatusSeeOther)
}

func (h *TeacherHandler) getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	token, _ := h.middleware.GetCSRFToken(cookie.Value)
	return token
}

func parseBulkNames(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	unique := make(map[string]struct{})
	var names []string
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, exists := unique[name]; exists {
			continue
		}
		unique[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

func (h *TeacherHandler) renderBulkCredentials(kids []models.Kid) string {
	if len(kids) == 0 {
		return `<div class="error-message">No child accounts were created.</div>`
	}

	var b strings.Builder
	b.WriteString(`<div class="credentials-display"><h3>Bulk Import Complete</h3><div class="credentials-box"><table class="table"><thead><tr><th>Name</th><th>Username</th><th>Password</th></tr></thead><tbody>`)
	for _, kid := range kids {
		b.WriteString(`<tr><td>` + template.HTMLEscapeString(kid.Name) + `</td><td><code>` + template.HTMLEscapeString(kid.Username) + `</code></td><td><code>` + template.HTMLEscapeString(kid.Password) + `</code></td></tr>`)
	}
	b.WriteString(`</tbody></table><p class="text-muted">Store these credentials securely for your class.</p></div></div>`)
	return b.String()
}
