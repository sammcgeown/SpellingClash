package handlers

import (
	"context"
	"log"
	"net/http"
	"spellingclash/internal/models"
	"spellingclash/internal/service"
	"spellingclash/internal/utils"
	"time"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	UserContextKey       ContextKey = "user"
	KidSessionContextKey ContextKey = "kid"
)

// Middleware holds dependencies for middleware functions
type Middleware struct {
	authService   *service.AuthService
	familyService *service.FamilyService
	csrfStore     *utils.CSRFTokenStore
	rateLimiter   *utils.RateLimiter
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(authService *service.AuthService, familyService *service.FamilyService) *Middleware {
	return &Middleware{
		authService:   authService,
		familyService: familyService,
		csrfStore:     utils.NewCSRFTokenStore(1 * time.Hour),
		rateLimiter:   utils.NewRateLimiter(100, 1*time.Minute), // 100 requests per minute
	}
}

// RequireReady is middleware that shows startup page if server is not ready
func RequireReady(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsReady() {
			ShowStartupStatus(w, r)
			return
		}
		next(w, r)
	}
}

// RequireAuth is middleware that requires a valid session
func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		user, err := m.authService.ValidateSession(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, utils.CreateDeleteCookie(r, "session_id"))
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// RequireKidAuth is middleware that requires a valid kid session
func (m *Middleware) RequireKidAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get kid session cookie
		cookie, err := r.Cookie("kid_session_id")
		if err != nil {
			http.Redirect(w, r, "/child/select", http.StatusSeeOther)
			return
		}

		// Validate kid session from database
		kidID, err := m.familyService.ValidateKidSession(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, utils.CreateDeleteCookie(r, "kid_session_id"))
			http.Redirect(w, r, "/child/select", http.StatusSeeOther)
			return
		}

		// Get kid from database
		kid, err := m.familyService.GetKid(kidID)
		if err != nil || kid == nil {
			// Clear invalid cookie
			http.SetCookie(w, utils.CreateDeleteCookie(r, "kid_session_id"))
			http.Redirect(w, r, "/child/select", http.StatusSeeOther)
			return
		}

		// Add kid to context
		ctx := context.WithValue(r.Context(), KidSessionContextKey, kid)
		next(w, r.WithContext(ctx))
	}
}

// RequireAdmin is middleware that requires a valid admin session
func (m *Middleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		user, err := m.authService.ValidateSession(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, utils.CreateDeleteCookie(r, "session_id"))
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Check if user is admin
		if !user.IsAdmin {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// Logging middleware logs HTTP requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Call next handler
		next.ServeHTTP(w, r)

		// Log request
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetKidFromContext retrieves the kid from the request context
func GetKidFromContext(ctx context.Context) *models.Kid {
	kid, ok := ctx.Value(KidSessionContextKey).(*models.Kid)
	if !ok {
		return nil
	}
	return kid
}

// RateLimit middleware limits requests per IP address
func (m *Middleware) RateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := utils.GetClientIP(r)

		if !m.rateLimiter.Allow(ip) {
			http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
			log.Printf("Rate limit exceeded for IP: %s", ip)
			return
		}

		next(w, r)
	}
}

// CSRFProtect middleware validates CSRF tokens on state-changing requests
func (m *Middleware) CSRFProtect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods
		if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" {
			// Get session ID
			cookie, err := r.Cookie("session_id")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get CSRF token from form/header
			token := r.FormValue("csrf_token")
			if token == "" {
				token = r.Header.Get("X-CSRF-Token")
			}

			if token == "" {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				log.Printf("CSRF token missing for %s %s", r.Method, r.URL.Path)
				return
			}

			// Validate token
			if !m.csrfStore.ValidateToken(cookie.Value, token) {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				log.Printf("Invalid CSRF token for %s %s", r.Method, r.URL.Path)
				return
			}
		}

		next(w, r)
	}
}

// GetCSRFToken retrieves or generates a CSRF token for the current session
func (m *Middleware) GetCSRFToken(sessionID string) (string, error) {
	// Try to get existing token
	if token, exists := m.csrfStore.GetToken(sessionID); exists {
		return token, nil
	}

	// Generate new token
	return m.csrfStore.GenerateToken(sessionID)
}
