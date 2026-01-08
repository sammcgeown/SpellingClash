package utils

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateEmail checks if an email address is valid
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ValidationError{Field: "email", Message: "email is required"}
	}
	if !emailRegex.MatchString(email) {
		return ValidationError{Field: "email", Message: "invalid email format"}
	}
	return nil
}

// ValidatePassword checks if a password meets requirements
func ValidatePassword(password string) error {
	if password == "" {
		return ValidationError{Field: "password", Message: "password is required"}
	}
	if len(password) < 8 {
		return ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}
	return nil
}

// ValidateName checks if a name is valid
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ValidationError{Field: "name", Message: "name is required"}
	}
	if len(name) < 2 {
		return ValidationError{Field: "name", Message: "name must be at least 2 characters"}
	}
	return nil
}
