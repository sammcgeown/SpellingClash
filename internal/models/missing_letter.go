package models

import "time"

// MissingLetterGame represents an active missing letter game session
type MissingLetterGame struct {
	ID              int64
	SessionID       int64
	KidID           int64
	WordID          int64
	Word            string
	MissingIndices  []int      // Indices of missing letters
	GuessedLetters  []string   // Letters guessed so far
	Attempts        int
	MaxAttempts     int
	IsWon           bool
	IsLost          bool
	StartedAt       time.Time
	CompletedAt     *time.Time
	PointsEarned    int
}

// MissingLetterGameState represents the current state of a missing letter game
type MissingLetterGameState struct {
	GameID          int64
	Word            string
	DisplayWord     string      // Word with blanks for missing letters
	MissingIndices  []int
	GuessedLetters  []string
	Attempts        int
	MaxAttempts     int
	IsWon           bool
	IsLost          bool
	IsComplete      bool
	RemainingWords  int
	CurrentWordIdx  int
	TotalWords      int
	PointsSoFar      int
	LastGuessCorrect *bool // nil if no guess yet, true/false for last guess result
	LastValidWordBonus *bool // true if last guess was a valid word but not the target
}

// MissingLetterSession represents a collection of missing letter games for a list
type MissingLetterSession struct {
	ID             int64
	KidID          int64
	SpellingListID int64
	StartedAt      time.Time
	CompletedAt    *time.Time
	TotalGames     int
	GamesWon       int
	TotalPoints    int
}
