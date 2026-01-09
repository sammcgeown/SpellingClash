package models

import "time"

// SpellingList represents a custom list of words to practice
type SpellingList struct {
	ID          int64
	FamilyID    *int64 // Nullable for public lists
	Name        string
	Description string
	CreatedBy   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsPublic    bool
}

// Word represents a word in a spelling list
type Word struct {
	ID              int64
	SpellingListID  int64
	WordText        string
	DifficultyLevel int // 1-5 scale
	AudioFilename   string
	Position        int
	CreatedAt       time.Time
}

// ListAssignment represents the assignment of a list to a kid
type ListAssignment struct {
	ID             int64
	SpellingListID int64
	KidID          int64
	AssignedAt     time.Time
	AssignedBy     int64
}

// ListWithWords combines a spelling list with its words
type ListWithWords struct {
	List  SpellingList
	Words []Word
}

// ListWithAssignments combines a spelling list with assignment info
type ListWithAssignments struct {
	List            SpellingList
	AssignedKids    []Kid
	TotalWords      int
	AvgDifficulty   float64
}

// ListSummary extends SpellingList with assignment count
type ListSummary struct {
	SpellingList
	AssignedKidCount int
}
