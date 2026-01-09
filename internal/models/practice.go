package models

import "time"

// PracticeSession represents a spelling practice session
type PracticeSession struct {
	ID             int64
	KidID          int64
	SpellingListID int64
	StartedAt      time.Time
	CompletedAt    *time.Time
	TotalWords     int
	CorrectWords   int
	PointsEarned   int
}

// WordAttempt represents a single word attempt in a practice session
type WordAttempt struct {
	ID                int64
	PracticeSessionID int64
	WordID            int64
	AttemptText       string
	IsCorrect         bool
	TimeTakenMs       int
	PointsEarned      int
	AttemptedAt       time.Time
}

// PracticeSessionWithDetails includes session data plus list and kid info
type PracticeSessionWithDetails struct {
	Session    PracticeSession
	ListName   string
	KidName    string
	Accuracy   float64 // Percentage of correct answers
}

// CurrentPracticeState represents the current state of an ongoing practice session
type CurrentPracticeState struct {
	SessionID       int64
	CurrentWordIdx  int
	TotalWords      int
	CorrectSoFar    int
	PointsSoFar     int
	RemainingWords  []Word
}

// PracticeState represents persisted practice state in the database
type PracticeState struct {
	KidID        int64
	SessionID    int64
	CurrentIndex int
	CorrectCount int
	TotalPoints  int
	StartTime    time.Time
	UpdatedAt    time.Time
}

