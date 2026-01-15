package repository

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"spellingclash/internal/database"
	"spellingclash/internal/models"
	"time"
)

// FamilyRepository handles database operations for families
type FamilyRepository struct {
	db *database.DB
}

// NewFamilyRepository creates a new family repository
func NewFamilyRepository(db *database.DB) *FamilyRepository {
	return &FamilyRepository{db: db}
}

// generateFamilyCode generates a random 8-character family code
func generateFamilyCode() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// CreateFamily creates a new family and adds the creator as a member
func (r *FamilyRepository) CreateFamily(name string, creatorUserID int64) (*models.Family, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Generate a unique family code
	familyCode := generateFamilyCode()

	// Create family
	query := "INSERT INTO families (name, family_code) VALUES (?, ?)"
	familyID, err := tx.ExecReturningID(query, name, familyCode)
	if err != nil {
		return nil, fmt.Errorf("failed to create family: %w", err)
	}

	// Add creator as admin member
	query = "INSERT INTO family_members (family_id, user_id, role) VALUES (?, ?, 'admin')"
	_, err = tx.Exec(query, familyID, creatorUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to add family member: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	family := &models.Family{
		ID:         familyID,
		Name:       name,
		FamilyCode: familyCode,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	return family, nil
}

// GetFamilyByID retrieves a family by ID
func (r *FamilyRepository) GetFamilyByID(familyID int64) (*models.Family, error) {
	query := "SELECT id, name, family_code, created_at, updated_at FROM families WHERE id = ?"
	family := &models.Family{}
	err := r.db.QueryRow(query, familyID).Scan(
		&family.ID,
		&family.Name,
		&family.FamilyCode,
		&family.CreatedAt,
		&family.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get family: %w", err)
	}

	return family, nil
}

// GetFamilyByCode retrieves a family by its unique code
func (r *FamilyRepository) GetFamilyByCode(code string) (*models.Family, error) {
	query := "SELECT id, name, family_code, created_at, updated_at FROM families WHERE family_code = ?"
	family := &models.Family{}
	err := r.db.QueryRow(query, code).Scan(
		&family.ID,
		&family.Name,
		&family.FamilyCode,
		&family.CreatedAt,
		&family.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get family by code: %w", err)
	}

	return family, nil
}

// GetUserFamilies retrieves all families a user belongs to
func (r *FamilyRepository) GetUserFamilies(userID int64) ([]models.Family, error) {
	query := `
		SELECT f.id, f.name, f.family_code, f.created_at, f.updated_at
		FROM families f
		INNER JOIN family_members fm ON f.id = fm.family_id
		WHERE fm.user_id = ?
		ORDER BY f.created_at DESC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query families: %w", err)
	}
	defer rows.Close()

	var families []models.Family
	for rows.Next() {
		var family models.Family
		if err := rows.Scan(&family.ID, &family.Name, &family.FamilyCode, &family.CreatedAt, &family.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan family: %w", err)
		}
		families = append(families, family)
	}

	return families, nil
}

// AddFamilyMember adds a user to a family
func (r *FamilyRepository) AddFamilyMember(familyID, userID int64, role string) error {
	query := "INSERT INTO family_members (family_id, user_id, role) VALUES (?, ?, ?)"
	_, err := r.db.Exec(query, familyID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to add family member: %w", err)
	}
	return nil
}

// IsFamilyMember checks if a user is a member of a family
func (r *FamilyRepository) IsFamilyMember(userID, familyID int64) (bool, error) {
	query := "SELECT COUNT(*) FROM family_members WHERE user_id = ? AND family_id = ?"
	var count int
	err := r.db.QueryRow(query, userID, familyID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check family membership: %w", err)
	}
	return count > 0, nil
}

// GetFamilyMembers retrieves all members of a family
func (r *FamilyRepository) GetFamilyMembers(familyID int64) ([]models.FamilyMember, []models.User, error) {
	query := `
		SELECT fm.id, fm.family_id, fm.user_id, fm.role, fm.joined_at,
		       u.id, u.email, u.name, u.created_at
		FROM family_members fm
		INNER JOIN users u ON fm.user_id = u.id
		WHERE fm.family_id = ?
		ORDER BY fm.joined_at ASC
	`
	rows, err := r.db.Query(query, familyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query family members: %w", err)
	}
	defer rows.Close()

	var members []models.FamilyMember
	var users []models.User
	for rows.Next() {
		var member models.FamilyMember
		var user models.User
		var userUpdatedAt time.Time // We'll ignore this for now
		if err := rows.Scan(
			&member.ID, &member.FamilyID, &member.UserID, &member.Role, &member.JoinedAt,
			&user.ID, &user.Email, &user.Name, &user.CreatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan family member: %w", err)
		}
		user.UpdatedAt = userUpdatedAt
		members = append(members, member)
		users = append(users, user)
	}

	return members, users, nil
}

// UpdateFamily updates a family's name
func (r *FamilyRepository) UpdateFamily(familyID int64, name string) error {
	query := "UPDATE families SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, name, familyID)
	if err != nil {
		return fmt.Errorf("failed to update family: %w", err)
	}
	return nil
}

// DeleteFamily deletes a family and all associated data
func (r *FamilyRepository) DeleteFamily(familyID int64) error {
	query := "DELETE FROM families WHERE id = ?"
	_, err := r.db.Exec(query, familyID)
	if err != nil {
		return fmt.Errorf("failed to delete family: %w", err)
	}
	return nil
}

// GetFamiliesByUser retrieves all families a user belongs to
func (r *FamilyRepository) GetFamiliesByUser(userID int64) ([]models.Family, error) {
	query := `
		SELECT f.id, f.name, f.family_code, f.created_at, f.updated_at
		FROM families f
		INNER JOIN family_members fm ON f.id = fm.family_id
		WHERE fm.user_id = ?
		ORDER BY f.name ASC
	`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query families: %w", err)
	}
	defer rows.Close()

	var families []models.Family
	for rows.Next() {
		var family models.Family
		if err := rows.Scan(
			&family.ID,
			&family.Name,
			&family.FamilyCode,
			&family.CreatedAt,
			&family.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan family: %w", err)
		}
		families = append(families, family)
	}

	return families, nil
}

// GetAllFamilies retrieves all families in the system
func (r *FamilyRepository) GetAllFamilies() ([]models.Family, error) {
	query := `
		SELECT id, name, family_code, created_at, updated_at
		FROM families
		ORDER BY name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query families: %w", err)
	}
	defer rows.Close()

	var families []models.Family
	for rows.Next() {
		var family models.Family
		if err := rows.Scan(
			&family.ID,
			&family.Name,
			&family.FamilyCode,
			&family.CreatedAt,
			&family.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan family: %w", err)
		}
		families = append(families, family)
	}

	return families, nil
}

// AddUserToFamily adds a user to a family
func (r *FamilyRepository) AddUserToFamily(userID, familyID int64) error {
	query := "INSERT INTO family_members (family_id, user_id, role) VALUES (?, ?, 'parent')"
	_, err := r.db.Exec(query, familyID, userID)
	if err != nil {
		return fmt.Errorf("failed to add user to family: %w", err)
	}
	return nil
}

// RemoveUserFromFamily removes a user from a family
func (r *FamilyRepository) RemoveUserFromFamily(userID, familyID int64) error {
	query := "DELETE FROM family_members WHERE user_id = ? AND family_id = ?"
	_, err := r.db.Exec(query, userID, familyID)
	if err != nil {
		return fmt.Errorf("failed to remove user from family: %w", err)
	}
	return nil
}
