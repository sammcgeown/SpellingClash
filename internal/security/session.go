package security

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

// GenerateSessionID creates a new UUID for session identification
func GenerateSessionID() string {
	return uuid.New().String()
}

// IsSecureRequest determines if the request is over HTTPS
// Checks TLS connection, X-Forwarded-Proto header (for reverse proxies), and URL scheme
func IsSecureRequest(r *http.Request) bool {
	// Direct TLS connection
	if r.TLS != nil {
		return true
	}

	// Behind reverse proxy (nginx, Caddy, load balancer, etc.)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return true
	}

	// Explicit HTTPS scheme
	if r.URL.Scheme == "https" {
		return true
	}

	return false
}

// CreateSessionCookie creates a session cookie with proper security flags
// The Secure flag is automatically set based on the request scheme (HTTPS detection)
func CreateSessionCookie(r *http.Request, name, value string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   IsSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	}
}

// CreateDeleteCookie creates a cookie for deletion with proper security flags
// The Secure flag is automatically set based on the request scheme (HTTPS detection)
func CreateDeleteCookie(r *http.Request, name string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   IsSecureRequest(r),
	}
}
