package handlers

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"spellingclash/internal/security"
	"spellingclash/internal/service"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService          *service.AuthService
	emailService         *service.EmailService
	templates            *template.Template
	oauthProviders       map[string]OAuthProvider
	oauthRedirectBaseURL string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, emailService *service.EmailService, templates *template.Template, oauthProviders map[string]OAuthProvider, oauthRedirectBaseURL string) *AuthHandler {
	return &AuthHandler{
		authService:          authService,
		emailService:         emailService,
		templates:            templates,
		oauthProviders:       oauthProviders,
		oauthRedirectBaseURL: oauthRedirectBaseURL,
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
		"Title":          "Login - WordClash",
		"OAuthProviders": h.oauthProviderViews(r),
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
			"Title":          "Login - WordClash",
			"Error":          "Invalid email or password",
			"Email":          email,
			"OAuthProviders": h.oauthProviderViews(r),
		}
		if err := h.templates.ExecuteTemplate(w, "login.tmpl", data); err != nil {
			log.Printf("Error rendering login template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Set session cookie
	http.SetCookie(w, security.CreateSessionCookie(r, "session_id", session.ID, session.ExpiresAt))

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
		"Title":          "Register - WordClash",
		"FamilyCode":     familyCode,
		"OAuthProviders": h.oauthProviderViews(r),
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
			"Title":          "Register - WordClash",
			"Error":          err.Error(),
			"Email":          email,
			"Name":           name,
			"FamilyCode":     familyCode,
			"OAuthProviders": h.oauthProviderViews(r),
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
	http.SetCookie(w, security.CreateSessionCookie(r, "session_id", session.ID, session.ExpiresAt))

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
	http.SetCookie(w, security.CreateDeleteCookie(r, "session_id"))

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

// ShowForgotPassword renders the forgot password page
func (h *AuthHandler) ShowForgotPassword(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Forgot Password - WordClash",
	}

	if err := h.templates.ExecuteTemplate(w, "forgot_password.tmpl", data); err != nil {
		log.Printf("Error rendering forgot password template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ForgotPassword handles forgot password form submission
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")

	// Request password reset
	ctx := context.Background()
	err := h.authService.RequestPasswordReset(ctx, h.emailService, email)

	// Always show success message (even if email doesn't exist - security best practice)
	data := map[string]interface{}{
		"Title":   "Password Reset Requested - WordClash",
		"Success": "If an account exists with that email, you will receive password reset instructions.",
	}

	if err != nil {
		log.Printf("Error requesting password reset: %v", err)
	}

	if err := h.templates.ExecuteTemplate(w, "forgot_password.tmpl", data); err != nil {
		log.Printf("Error rendering forgot password template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ShowResetPassword renders the reset password page
func (h *AuthHandler) ShowResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	if token == "" {
		http.Redirect(w, r, "/auth/forgot-password", http.StatusSeeOther)
		return
	}

	// Verify token is valid
	valid, err := h.authService.ValidatePasswordResetToken(token)
	if err != nil || !valid {
		data := map[string]interface{}{
			"Title": "Reset Password - WordClash",
			"Error": "This password reset link is invalid or has expired. Please request a new one.",
		}
		if err := h.templates.ExecuteTemplate(w, "reset_password.tmpl", data); err != nil {
			log.Printf("Error rendering reset password template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	data := map[string]interface{}{
		"Title": "Reset Password - WordClash",
		"Token": token,
	}

	if err := h.templates.ExecuteTemplate(w, "reset_password.tmpl", data); err != nil {
		log.Printf("Error rendering reset password template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ResetPassword handles reset password form submission
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")

	// Attempt password reset
	err := h.authService.ResetPassword(token, password)
	if err != nil {
		data := map[string]interface{}{
			"Title": "Reset Password - WordClash",
			"Token": token,
			"Error": err.Error(),
		}
		if err := h.templates.ExecuteTemplate(w, "reset_password.tmpl", data); err != nil {
			log.Printf("Error rendering reset password template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Success - redirect to login
	data := map[string]interface{}{
		"Title":          "Login - WordClash",
		"Success":        "Your password has been reset successfully. Please log in with your new password.",
		"OAuthProviders": h.oauthProviderViews(r),
	}

	if err := h.templates.ExecuteTemplate(w, "login.tmpl", data); err != nil {
		log.Printf("Error rendering login template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
