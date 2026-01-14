package models

import "time"

// HangmanGame represents an active hangman game session
type HangmanGame struct {
	ID             int64
	KidID          int64
	SpellingListID int64
	Word           string
	GuessedLetters []string
	WrongGuesses   int
	MaxWrongGuesses int
	IsWon          bool
	IsLost         bool
	StartedAt      time.Time
	CompletedAt    *time.Time
	PointsEarned   int
}

// HangmanGameState represents the current state of a hangman game
type HangmanGameState struct {
	GameID         int64
	Word           string
	MaskedWord     string
	GuessedLetters []string
	WrongGuesses   int
	MaxWrongGuesses int
	IsWon          bool
	IsLost         bool
	IsComplete     bool
	RemainingWords int
	CurrentWordIdx int
	TotalWords     int
	PointsSoFar    int
}

// HangmanSession represents a collection of hangman games for a list
type HangmanSession struct {
	ID             int64
	KidID          int64
	SpellingListID int64
	StartedAt      time.Time
	CompletedAt    *time.Time
	TotalGames     int
	GamesWon       int
	TotalPoints    int
}
