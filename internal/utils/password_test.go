package utils

import (
	"testing"
)

func TestGenerateKidPassword(t *testing.T) {
	tests := []struct {
		name        string
		iterations  int
		minLength   int
		maxLength   int
		shouldMatch bool
	}{
		{
			name:       "generates password of correct length",
			iterations: 100,
			minLength:  4,
			maxLength:  4,
		},
		{
			name:        "generates unique passwords",
			iterations:  10,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passwords := make(map[string]bool)
			for i := 0; i < tt.iterations; i++ {
				password, _ := GenerateKidPassword()

				// Check length
				if len(password) < tt.minLength || len(password) > tt.maxLength {
					t.Errorf("password length %d not in range [%d, %d]", len(password), tt.minLength, tt.maxLength)
				}

				// Check uniqueness
				if !tt.shouldMatch {
					if passwords[password] {
						t.Errorf("duplicate password generated: %s", password)
					}
					passwords[password] = true
				}
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	password := "testPassword123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == "" {
		t.Error("HashPassword() returned empty string")
	}

	if hash == password {
		t.Error("HashPassword() returned unhashed password")
	}

	// Test same password produces different hashes (due to salt)
	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	if hash == hash2 {
		t.Error("HashPassword() should produce different hashes due to salt")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "mySecurePassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongPassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckPassword(tt.password, tt.hash)
			if result != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", result, tt.want)
			}
		})
	}
}
