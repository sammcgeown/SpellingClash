package utils

import (
	"github.com/google/uuid"
)

// GenerateSessionID creates a new UUID for session identification
func GenerateSessionID() string {
	return uuid.New().String()
}
