package repository

import (
	"database/sql"
	"fmt"
	"time"
	"wordclash/internal/models"
)

// KidRepository handles database operations for kids
type KidRepository struct {
	db *sql.DB
}

// NewKidRepository creates a new kid repository
func NewKidRepository(db *sql.DB) *KidRepository {
	return &KidRepository{db: db}
}

// CreateKid creates a new kid profile
func (r *KidRepository) CreateKid(familyID int64, name, avatarColor string) (*models.Kid, error) {
	query := "INSERT INTO kids (family_id, name, avatar_color) VALUES (?, ?, ?)"
	result, err := r.db.Exec(query, familyID, name, avatarColor)
	if err != nil {
		return nil, fmt.Errorf("failed to create kid: %w", err)
	}

	kidID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get kid ID: %w", err)
	}

	kid := &models.Kid{
		ID:          kidID,
		FamilyID:    familyID,
		Name:        name,
		AvatarColor: avatarColor,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return kid, nil
}

// GetKidByID retrieves a kid by ID
func (r *KidRepository) GetKidByID(kidID int64) (*models.Kid, error) {
	query := "SELECT id, family_id, name, avatar_color, created_at, updated_at FROM kids WHERE id = ?"
	kid := &models.Kid{}
	err := r.db.QueryRow(query, kidID).Scan(
		&kid.ID,
		&kid.FamilyID,
		&kid.Name,
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
func (r *KidRepository) GetFamilyKids(familyID int64) ([]models.Kid, error) {
	query := `
		SELECT id, family_id, name, avatar_color, created_at, updated_at
		FROM kids
		WHERE family_id = ?
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query kids: %w", err)
	}
	defer rows.Close()

	var kids []models.Kid
	for rows.Next() {
		var kid models.Kid
		if err := rows.Scan(
			&kid.ID,
			&kid.FamilyID,
			&kid.Name,
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
		SELECT id, family_id, name, avatar_color, created_at, updated_at
		FROM kids
		ORDER BY name ASC
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
			&kid.FamilyID,
			&kid.Name,
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

// DeleteKid deletes a kid profile
func (r *KidRepository) DeleteKid(kidID int64) error {
	query := "DELETE FROM kids WHERE id = ?"
	_, err := r.db.Exec(query, kidID)
	if err != nil {
		return fmt.Errorf("failed to delete kid: %w", err)
	}
	return nil
}
