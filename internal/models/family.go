package models

import "time"

// Family represents a group of parents managing kids together
type Family struct {
	ID         int64
	Name       string
	FamilyCode string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// FamilyMember represents the relationship between a user and a family
type FamilyMember struct {
	ID       int64
	FamilyID int64
	UserID   int64
	Role     string // 'parent' or 'admin'
	JoinedAt time.Time
}

// FamilyWithMembers combines a family with its member information
type FamilyWithMembers struct {
	Family  Family
	Members []FamilyMember
	Users   []User // Associated user details
}
