package models

import "time"

// Kid represents a child profile in the system
type Kid struct {
	ID          int64
	FamilyCode  string
	Name        string
	Username    string // Randomly generated username (e.g., "happy-dragon")
	Password    string // Randomly generated 4-character password
	AvatarColor string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// KidWithStats combines a kid with their statistics
type KidWithStats struct {
	Kid                  Kid
	TotalPractices       int
	CurrentDailyStreak   int
	LongestDailyStreak   int
	CurrentCorrectStreak int
	TotalPoints          int
	AssignedListsCount   int
}

// KidWithLists combines a kid with their assigned spelling lists
type KidWithLists struct {
	Kid           Kid
	AssignedLists []SpellingList
}

// KidStats represents overall practice statistics for a kid
type KidStats struct {
	TotalSessions         int
	TotalWordsPracticed   int
	TotalCorrect          int
	TotalPoints           int
	UniqueWordsAttempted  int
	OverallAccuracy       float64
}
