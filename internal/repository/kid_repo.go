package repository

import (
	"database/sql"
	"fmt"
	"spellingclash/internal/database"
	"spellingclash/internal/models"
	"time"
)

// KidRepository handles database operations for kids
type KidRepository struct {
	db *database.DB
}

// NewKidRepository creates a new kid repository
func NewKidRepository(db *database.DB) *KidRepository {
	return &KidRepository{db: db}
}

// CreateKid creates a new kid profile
func (r *KidRepository) CreateKid(familyCode, name, username, password, avatarColor string) (*models.Kid, error) {
	query := "INSERT INTO kids (family_code, name, username, password, avatar_color) VALUES (?, ?, ?, ?, ?)"
	kidID, err := r.db.ExecReturningID(query, familyCode, name, username, password, avatarColor)
	if err != nil {
		return nil, fmt.Errorf("failed to create kid: %w", err)
	}

	kid := &models.Kid{
		ID:          kidID,
		FamilyCode:  familyCode,
		Name:        name,
		Username:    username,
		Password:    password,
		AvatarColor: avatarColor,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return kid, nil
}

// GetKidByID retrieves a kid by ID
func (r *KidRepository) GetKidByID(kidID int64) (*models.Kid, error) {
	query := "SELECT id, family_code, name, username, password, avatar_color, created_at, updated_at FROM kids WHERE id = ?"
	kid := &models.Kid{}
	err := r.db.QueryRow(query, kidID).Scan(
		&kid.ID,
		&kid.FamilyCode,
		&kid.Name,
		&kid.Username,
		&kid.Password,
		&kid.AvatarColor,
		&kid.CreatedAt,
		&kid.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get kid: %w", err)
	}

	return kid, nil
}

// GetKidByUsername retrieves a kid by username
func (r *KidRepository) GetKidByUsername(username string) (*models.Kid, error) {
	query := "SELECT id, family_code, name, username, password, avatar_color, created_at, updated_at FROM kids WHERE username = ?"
	kid := &models.Kid{}
	err := r.db.QueryRow(query, username).Scan(
		&kid.ID,
		&kid.FamilyCode,
		&kid.Name,
		&kid.Username,
		&kid.Password,
		&kid.AvatarColor,
		&kid.CreatedAt,
		&kid.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get kid: %w", err)
	}

	return kid, nil
}

// GetFamilyKids retrieves all kids in a family
func (r *KidRepository) GetFamilyKids(familyCode string) ([]models.Kid, error) {
	query := `
		SELECT id, family_code, name, username, password, avatar_color, created_at, updated_at
		FROM kids
		WHERE family_code = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query, familyCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query kids: %w", err)
	}
	defer rows.Close()

	var kids []models.Kid
	for rows.Next() {
		var kid models.Kid
		if err := rows.Scan(
			&kid.ID,
			&kid.FamilyCode,
			&kid.Name,
			&kid.Username,
			&kid.Password,
			&kid.AvatarColor,
			&kid.CreatedAt,
			&kid.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kid: %w", err)
		}
		kids = append(kids, kid)
	}

	return kids, nil
}

// GetAllKids retrieves all kids from all families
func (r *KidRepository) GetAllKids() ([]models.Kid, error) {
	query := `
		SELECT id, family_code, name, username, password, avatar_color, created_at, updated_at
		FROM kids
		ORDER BY username ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query kids: %w", err)
	}
	defer rows.Close()

	var kids []models.Kid
	for rows.Next() {
		var kid models.Kid
		if err := rows.Scan(
			&kid.ID,
			&kid.FamilyCode,
			&kid.Name,
			&kid.Username,
			&kid.Password,
			&kid.AvatarColor,
			&kid.CreatedAt,
			&kid.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan kid: %w", err)
		}
		kids = append(kids, kid)
	}

	return kids, nil
}

// UpdateKid updates a kid's information
func (r *KidRepository) UpdateKid(kidID int64, name, avatarColor string) error {
	query := "UPDATE kids SET name = ?, avatar_color = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, name, avatarColor, kidID)
	if err != nil {
		return fmt.Errorf("failed to update kid: %w", err)
	}
	return nil
}
// UpdateKidPassword updates a kid's password
func (r *KidRepository) UpdateKidPassword(kidID int64, password string) error {
	query := "UPDATE kids SET password = ? WHERE id = ?"
	_, err := r.db.Exec(query, password, kidID)
	if err != nil {
		return fmt.Errorf("failed to update kid password: %w", err)
	}
	return nil
}
// DeleteKid deletes a kid profile
func (r *KidRepository) DeleteKid(kidID int64) error {
	query := "DELETE FROM kids WHERE id = ?"
	_, err := r.db.Exec(query, kidID)
	if err != nil {
		return fmt.Errorf("failed to delete kid: %w", err)
	}
	return nil
}

// CreateKidSession creates a new session for a kid
func (r *KidRepository) CreateKidSession(sessionID string, kidID int64, expiresAt time.Time) error {
	query := `
		INSERT INTO kid_sessions (id, kid_id, expires_at)
		VALUES (?, ?, ?)
	`
	_, err := r.db.Exec(query, sessionID, kidID, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create kid session: %w", err)
	}
	return nil
}

// GetKidSession retrieves a kid session by ID
func (r *KidRepository) GetKidSession(sessionID string) (int64, error) {
	query := `
		SELECT kid_id, expires_at
		FROM kid_sessions
		WHERE id = ?
	`
	var kidID int64
	var expiresAt time.Time
	err := r.db.QueryRow(query, sessionID).Scan(&kidID, &expiresAt)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("session not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get kid session: %w", err)
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		// Clean up expired session
		_ = r.DeleteKidSession(sessionID)
		return 0, fmt.Errorf("session expired")
	}

	return kidID, nil
}

// DeleteKidSession removes a kid session from the database
func (r *KidRepository) DeleteKidSession(sessionID string) error {
	query := "DELETE FROM kid_sessions WHERE id = ?"
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete kid session: %w", err)
	}
	return nil
}

// DeleteExpiredKidSessions removes all expired kid sessions
func (r *KidRepository) DeleteExpiredKidSessions() error {
	query := "DELETE FROM kid_sessions WHERE expires_at < ?"
	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired kid sessions: %w", err)
	}
	return nil
}
