package models

import "time"

type Invitation struct {
	ID         int64
	Code       string
	Email      string
	InvitedBy  int64
	CreatedAt  time.Time
	UsedAt     *time.Time
	UsedBy     *int64
	ExpiresAt  time.Time
	InviterName string // Populated via JOIN
}

func (i *Invitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

func (i *Invitation) IsUsed() bool {
	return i.UsedAt != nil
}

func (i *Invitation) IsValid() bool {
	return !i.IsExpired() && !i.IsUsed()
}
