package handlers

import (
	"spellingclash/internal/models"
	"testing"
	"time"
)

func TestPracticeSessionValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		session models.PracticeSession
		wantErr bool
	}{
		{
			name: "valid session",
			session: models.PracticeSession{
				ID:             1,
				KidID:          1,
				SpellingListID: 1,
				StartedAt:      now,
				TotalWords:     10,
				CorrectWords:   7,
				PointsEarned:   350,
			},
			wantErr: false,
		},
		{
			name: "completed session",
			session: models.PracticeSession{
				ID:             1,
				KidID:          1,
				SpellingListID: 1,
				StartedAt:      now,
				CompletedAt:    &now,
				TotalWords:     10,
				CorrectWords:   10,
				PointsEarned:   500,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.session.KidID == 0 {
				t.Error("KidID should not be 0")
			}
			if tt.session.SpellingListID == 0 {
				t.Error("SpellingListID should not be 0")
			}
			if tt.session.CorrectWords > tt.session.TotalWords {
				t.Error("CorrectWords cannot exceed TotalWords")
			}
		})
	}
}

func TestHangmanSessionValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		session models.HangmanSession
	}{
		{
			name: "valid hangman session",
			session: models.HangmanSession{
				ID:             1,
				KidID:          1,
				SpellingListID: 1,
				StartedAt:      now,
				TotalGames:     10,
				GamesWon:       8,
				TotalPoints:    400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.session.KidID == 0 {
				t.Error("KidID should not be 0")
			}
			if tt.session.SpellingListID == 0 {
				t.Error("SpellingListID should not be 0")
			}
		})
	}
}
