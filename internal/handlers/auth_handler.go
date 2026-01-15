package handlers

import (
	"html/template"
	"log"
	"net/http"
	"spellingclash/internal/service"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *service.AuthService
	templates   *template.Template
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, templates *template.Template) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		templates:   templates,
	}
}

// ShowLogin renders the login page
func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	if cookie, err := r.Cookie("session_id"); err == nil {
		if _, err := h.authService.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
			return
		}
	}

	data := map[string]interface{}{
		"Title": "Login - WordClash",
	}

	if err := h.templates.ExecuteTemplate(w, "login.tmpl", data); err != nil {
		log.Printf("Error rendering login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Login handles login form submission
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Attempt login
	session, _, err := h.authService.Login(email, password)
	if err != nil {
		// Re-render login with error
		data := map[string]interface{}{
			"Title": "Login - WordClash",
			"Error": "Invalid email or password",
			"Email": email,
		}
		if err := h.templates.ExecuteTemplate(w, "login.tmpl", data); err != nil {
			log.Printf("Error rendering login template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
}

// ShowRegister renders the registration page
func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	if cookie, err := r.Cookie("session_id"); err == nil {
		if _, err := h.authService.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
			return
		}
	}

	// Get family_code from query parameter if present
	familyCode := r.URL.Query().Get("family_code")

	data := map[string]interface{}{
		"Title":      "Register - WordClash",
		"FamilyCode": familyCode,
	}

	if err := h.templates.ExecuteTemplate(w, "register.tmpl", data); err != nil {
		log.Printf("Error rendering register template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Register handles registration form submission
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	name := r.FormValue("name")
	familyCode := r.FormValue("family_code")

	// Attempt registration
	_, err := h.authService.Register(email, password, name, familyCode)
	if err != nil {
		// Re-render register with error
		data := map[string]interface{}{
			"Title":      "Register - WordClash",
			"Error":      err.Error(),
			"Email":      email,
			"Name":       name,
			"FamilyCode": familyCode,
		}
		if err := h.templates.ExecuteTemplate(w, "register.tmpl", data); err != nil {
			log.Printf("Error rendering register template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Auto-login after registration
	session, _, err := h.authService.Login(email, password)
	if err != nil {
		// Registration succeeded but login failed - redirect to login
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to dashboard
	http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
}

// Logout handles logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err == nil {
		// Delete session from database
		_ = h.authService.Logout(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Home renders the home page
func (h *AuthHandler) Home(w http.ResponseWriter, r *http.Request) {
	// Check if logged in
	if cookie, err := r.Cookie("session_id"); err == nil {
		if _, err := h.authService.ValidateSession(cookie.Value); err == nil {
			http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
			return
		}
	}

	// Redirect to login for now (we can create a proper landing page later)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
