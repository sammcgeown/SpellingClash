package repository

import (
	"fmt"
	"spellingclash/internal/database"
	"spellingclash/internal/models"
)

// TeacherKidRepository handles teacher-to-child relationships.
type TeacherKidRepository struct {
	db *database.DB
}

// NewTeacherKidRepository creates a new teacher-kid repository.
func NewTeacherKidRepository(db *database.DB) *TeacherKidRepository {
	return &TeacherKidRepository{db: db}
}

// LinkTeacherToKid creates a teacher-child relationship.
func (r *TeacherKidRepository) LinkTeacherToKid(teacherUserID, kidID int64) error {
	query := `
		INSERT INTO teacher_kid_relationships (teacher_user_id, kid_id)
		VALUES (?, ?)
	`
	if _, err := r.db.Exec(query, teacherUserID, kidID); err != nil {
		return fmt.Errorf("failed to link teacher to kid: %w", err)
	}
	return nil
}

// IsTeacherLinkedToKid checks if a teacher is linked to the given kid.
func (r *TeacherKidRepository) IsTeacherLinkedToKid(teacherUserID, kidID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM teacher_kid_relationships
		WHERE teacher_user_id = ? AND kid_id = ?
	`
	var count int
	if err := r.db.QueryRow(query, teacherUserID, kidID).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check teacher-kid link: %w", err)
	}
	return count > 0, nil
}

// GetTeacherKids retrieves all kids linked to a teacher.
func (r *TeacherKidRepository) GetTeacherKids(teacherUserID int64) ([]models.Kid, error) {
	query := `
		SELECT k.id, k.family_code, k.name, k.username, k.password, k.avatar_color, k.created_at, k.updated_at
		FROM teacher_kid_relationships tkr
		INNER JOIN kids k ON k.id = tkr.kid_id
		WHERE tkr.teacher_user_id = ?
		ORDER BY k.created_at DESC
	`

	rows, err := r.db.Query(query, teacherUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to query teacher kids: %w", err)
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
			return nil, fmt.Errorf("failed to scan teacher kid: %w", err)
		}
		kids = append(kids, kid)
	}

	return kids, nil
}
