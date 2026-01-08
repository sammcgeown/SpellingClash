package models

import "time"

// Kid represents a child profile in the system
type Kid struct {
	ID          int64
	FamilyID    int64
	Name        string
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
