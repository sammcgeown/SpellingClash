package repository

import (
	"crypto/rand"
	"encoding/hex"
	"spellingclash/internal/database"
	"spellingclash/internal/models"
	"time"
)

type InvitationRepository struct {
	db *database.DB
}

func NewInvitationRepository(db *database.DB) *InvitationRepository {
	return &InvitationRepository{db: db}
}

// GenerateInvitationCode generates a random invitation code
func (r *InvitationRepository) GenerateInvitationCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateInvitation creates a new invitation
func (r *InvitationRepository) CreateInvitation(email string, invitedBy int64, expiresAt time.Time) (*models.Invitation, error) {
	code, err := r.GenerateInvitationCode()
	if err != nil {
		return nil, err
	}

	query := `INSERT INTO invitations (code, email, invited_by, expires_at) VALUES (?, ?, ?, ?)`
	id, err := r.db.ExecReturningID(query, code, email, invitedBy, expiresAt)
	if err != nil {
		return nil, err
	}

	return &models.Invitation{
		ID:        id,
		Code:      code,
		Email:     email,
		InvitedBy: invitedBy,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}, nil
}

// GetInvitationByCode retrieves an invitation by code
func (r *InvitationRepository) GetInvitationByCode(code string) (*models.Invitation, error) {
	query := `
		SELECT i.id, i.code, i.email, i.invited_by, i.created_at, i.used_at, i.used_by, i.expires_at, u.name
		FROM invitations i
		LEFT JOIN users u ON i.invited_by = u.id
		WHERE i.code = ?
	`
	
	var inv models.Invitation
	var usedAt, usedBy interface{}
	
	err := r.db.QueryRow(query, code).Scan(
		&inv.ID, &inv.Code, &inv.Email, &inv.InvitedBy,
		&inv.CreatedAt, &usedAt, &usedBy, &inv.ExpiresAt, &inv.InviterName,
	)
	if err != nil {
		return nil, err
	}

	if usedAt != nil {
		t := usedAt.(time.Time)
		inv.UsedAt = &t
	}
	if usedBy != nil {
		id := usedBy.(int64)
		inv.UsedBy = &id
	}

	return &inv, nil
}

// MarkInvitationUsed marks an invitation as used
func (r *InvitationRepository) MarkInvitationUsed(code string, userId int64) error {
	query := `UPDATE invitations SET used_at = ?, used_by = ? WHERE code = ?`
	_, err := r.db.Exec(query, time.Now(), userId, code)
	return err
}

// GetAllInvitations retrieves all invitations (for admin view)
func (r *InvitationRepository) GetAllInvitations() ([]models.Invitation, error) {
	query := `
		SELECT i.id, i.code, i.email, i.invited_by, i.created_at, i.used_at, i.used_by, i.expires_at, u.name
		FROM invitations i
		LEFT JOIN users u ON i.invited_by = u.id
		ORDER BY i.created_at DESC
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []models.Invitation
	for rows.Next() {
		var inv models.Invitation
		var usedAt, usedBy interface{}
		
		err := rows.Scan(
			&inv.ID, &inv.Code, &inv.Email, &inv.InvitedBy,
			&inv.CreatedAt, &usedAt, &usedBy, &inv.ExpiresAt, &inv.InviterName,
		)
		if err != nil {
			return nil, err
		}

		if usedAt != nil {
			t := usedAt.(time.Time)
			inv.UsedAt = &t
		}
		if usedBy != nil {
			id := usedBy.(int64)
			inv.UsedBy = &id
		}

		invitations = append(invitations, inv)
	}

	return invitations, nil
}

// DeleteInvitation deletes an invitation by ID
func (r *InvitationRepository) DeleteInvitation(id int64) error {
	query := `DELETE FROM invitations WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}
