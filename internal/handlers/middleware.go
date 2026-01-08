package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"wordclash/internal/models"
	"wordclash/internal/service"
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
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(authService *service.AuthService, familyService *service.FamilyService) *Middleware {
	return &Middleware{
		authService:   authService,
		familyService: familyService,
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
			http.SetCookie(w, &http.Cookie{
				Name:     "session_id",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
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
			http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
			return
		}

		// Parse kid ID from cookie
		kidID, err := parseKidID(cookie.Value)
		if err != nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "kid_session_id",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
			return
		}

		// Get kid from database
		kid, err := m.familyService.GetKid(kidID)
		if err != nil || kid == nil {
			// Clear invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "kid_session_id",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/kid/select", http.StatusSeeOther)
			return
		}

		// Add kid to context
		ctx := context.WithValue(r.Context(), KidSessionContextKey, kid)
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

// parseKidID parses a kid ID from a string
func parseKidID(s string) (int64, error) {
	var kidID int64
	_, err := fmt.Sscanf(s, "%d", &kidID)
	return kidID, err
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}
