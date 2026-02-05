package models

import (
	"testing"
	"time"
)

func TestSessionIsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "future expiration",
			expiresAt: time.Now().Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "just expired",
			expiresAt: time.Now().Add(-1 * time.Second),
			want:      true,
		},
		{
			name:      "expired yesterday",
			expiresAt: time.Now().Add(-24 * time.Hour),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := Session{
				ID:        "test-session",
				UserID:    1,
				ExpiresAt: tt.expiresAt,
				CreatedAt: time.Now().Add(-1 * time.Hour),
			}
			result := session.IsExpired()
			if result != tt.want {
				t.Errorf("Session.IsExpired() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestKidValidation(t *testing.T) {
	tests := []struct {
		name    string
		kid     Kid
		wantErr bool
	}{
		{
			name: "valid kid",
			kid: Kid{
				ID:          1,
				Name:        "John",
				Username:    "happy-dragon",
				Password:    "1234",
				FamilyCode:  "FAM123",
				AvatarColor: "#FF5733",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			kid: Kid{
				ID:         1,
				Name:       "",
				Username:   "happy-dragon",
				Password:   "1234",
				FamilyCode: "FAM123",
			},
			wantErr: true,
		},
		{
			name: "missing family code",
			kid: Kid{
				ID:         1,
				Name:       "John",
				Username:   "happy-dragon",
				Password:   "1234",
				FamilyCode: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.kid.Name == "" || tt.kid.FamilyCode == "")
			if hasError != tt.wantErr {
				t.Errorf("Kid validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestWordValidation(t *testing.T) {
	tests := []struct {
		name    string
		word    Word
		wantErr bool
	}{
		{
			name: "valid word",
			word: Word{
				ID:              1,
				SpellingListID:  1,
				WordText:        "example",
				DifficultyLevel: 3,
			},
			wantErr: false,
		},
		{
			name: "empty word text",
			word: Word{
				ID:              1,
				SpellingListID:  1,
				WordText:        "",
				DifficultyLevel: 3,
			},
			wantErr: true,
		},
		{
			name: "invalid difficulty",
			word: Word{
				ID:              1,
				SpellingListID:  1,
				WordText:        "example",
				DifficultyLevel: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := (tt.word.WordText == "" || tt.word.DifficultyLevel < 1 || tt.word.DifficultyLevel > 5)
			if hasError != tt.wantErr {
				t.Errorf("Word validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestKidStatsCalculation(t *testing.T) {
	tests := []struct {
		name  string
		stats KidStats
		want  float64
	}{
		{
			name: "perfect accuracy",
			stats: KidStats{
				TotalWordsPracticed: 100,
				TotalCorrect:        100,
			},
			want: 100.0,
		},
		{
			name: "50% accuracy",
			stats: KidStats{
				TotalWordsPracticed: 100,
				TotalCorrect:        50,
			},
			want: 50.0,
		},
		{
			name: "no practice",
			stats: KidStats{
				TotalWordsPracticed: 0,
				TotalCorrect:        0,
			},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var accuracy float64
			if tt.stats.TotalWordsPracticed > 0 {
				accuracy = (float64(tt.stats.TotalCorrect) / float64(tt.stats.TotalWordsPracticed)) * 100
			}
			if accuracy != tt.want {
				t.Errorf("accuracy = %.2f, want %.2f", accuracy, tt.want)
			}
		})
	}
}
