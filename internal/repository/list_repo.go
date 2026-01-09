package repository

import (
	"database/sql"
	"fmt"
	"time"
	"wordclash/internal/models"
)

// ListRepository handles database operations for spelling lists and words
type ListRepository struct {
	db *sql.DB
}

// NewListRepository creates a new list repository
func NewListRepository(db *sql.DB) *ListRepository {
	return &ListRepository{db: db}
}

// CreateList creates a new spelling list
func (r *ListRepository) CreateList(familyID int64, name, description string, createdBy int64) (*models.SpellingList, error) {
	query := "INSERT INTO spelling_lists (family_id, name, description, created_by, is_public) VALUES (?, ?, ?, ?, 0)"
	result, err := r.db.Exec(query, familyID, name, description, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create list: %w", err)
	}

	listID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get list ID: %w", err)
	}

	list := &models.SpellingList{
		ID:          listID,
		FamilyID:    &familyID,
		Name:        name,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsPublic:    false,
	}

	return list, nil
}

// CreatePublicList creates a public spelling list (not tied to any family)
func (r *ListRepository) CreatePublicList(name, description string) (*models.SpellingList, error) {
	query := "INSERT INTO spelling_lists (family_id, name, description, created_by, is_public) VALUES (NULL, ?, ?, 1, 1)"
	result, err := r.db.Exec(query, name, description)
	if err != nil {
		return nil, fmt.Errorf("failed to create public list: %w", err)
	}

	listID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get list ID: %w", err)
	}

	list := &models.SpellingList{
		ID:          listID,
		FamilyID:    nil,
		Name:        name,
		Description: description,
		CreatedBy:   1, // System/Admin user
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsPublic:    true,
	}

	return list, nil
}

// PublicListExists checks if a public list with the given name already exists
func (r *ListRepository) PublicListExists(name string) (bool, error) {
	query := "SELECT COUNT(*) FROM spelling_lists WHERE name = ? AND is_public = 1"
	var count int
	err := r.db.QueryRow(query, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if public list exists: %w", err)
	}
	return count > 0, nil
}

// GetListByID retrieves a spelling list by ID
func (r *ListRepository) GetListByID(listID int64) (*models.SpellingList, error) {
	query := `
		SELECT id, family_id, name, description, created_by, created_at, updated_at, is_public
		FROM spelling_lists
		WHERE id = ?
	`
	list := &models.SpellingList{}
	err := r.db.QueryRow(query, listID).Scan(
		&list.ID,
		&list.FamilyID,
		&list.Name,
		&list.Description,
		&list.CreatedBy,
		&list.CreatedAt,
		&list.UpdatedAt,
		&list.IsPublic,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get list: %w", err)
	}

	return list, nil
}

// GetFamilyLists retrieves all spelling lists for a family
func (r *ListRepository) GetFamilyLists(familyID int64) ([]models.SpellingList, error) {
	query := `
		SELECT id, family_id, name, description, created_by, created_at, updated_at, is_public
		FROM spelling_lists
		WHERE family_id = ?
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lists: %w", err)
	}
	defer rows.Close()

	var lists []models.SpellingList
	for rows.Next() {
		var list models.SpellingList
		if err := rows.Scan(
			&list.ID,
			&list.FamilyID,
			&list.Name,
			&list.Description,
			&list.CreatedBy,
			&list.CreatedAt,
			&list.UpdatedAt,
			&list.IsPublic,
		); err != nil {
			return nil, fmt.Errorf("failed to scan list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, nil
}

// GetPublicLists retrieves all public spelling lists available to everyone
func (r *ListRepository) GetPublicLists() ([]models.SpellingList, error) {
	query := `
		SELECT id, family_id, name, description, created_by, created_at, updated_at, is_public
		FROM spelling_lists
		WHERE is_public = 1
		ORDER BY name ASC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query public lists: %w", err)
	}
	defer rows.Close()

	var lists []models.SpellingList
	for rows.Next() {
		var list models.SpellingList
		if err := rows.Scan(
			&list.ID,
			&list.FamilyID,
			&list.Name,
			&list.Description,
			&list.CreatedBy,
			&list.CreatedAt,
			&list.UpdatedAt,
			&list.IsPublic,
		); err != nil {
			return nil, fmt.Errorf("failed to scan public list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, nil
}

// UpdateList updates a spelling list's name and description
func (r *ListRepository) UpdateList(listID int64, name, description string) error {
	query := "UPDATE spelling_lists SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := r.db.Exec(query, name, description, listID)
	if err != nil {
		return fmt.Errorf("failed to update list: %w", err)
	}
	return nil
}

// DeleteList deletes a spelling list and all associated data
func (r *ListRepository) DeleteList(listID int64) error {
	query := "DELETE FROM spelling_lists WHERE id = ?"
	_, err := r.db.Exec(query, listID)
	if err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
	}
	return nil
}

// AddWord adds a word to a spelling list
func (r *ListRepository) AddWord(listID int64, wordText string, difficulty, position int, definition string) (*models.Word, error) {
	var query string
	var result sql.Result
	var err error
	
	if definition != "" {
		query = "INSERT INTO words (spelling_list_id, word_text, difficulty_level, position, definition) VALUES (?, ?, ?, ?, ?)"
		result, err = r.db.Exec(query, listID, wordText, difficulty, position, definition)
	} else {
		query = "INSERT INTO words (spelling_list_id, word_text, difficulty_level, position) VALUES (?, ?, ?, ?)"
		result, err = r.db.Exec(query, listID, wordText, difficulty, position)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to add word: %w", err)
	}

	wordID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get word ID: %w", err)
	}

	word := &models.Word{
		ID:              wordID,
		SpellingListID:  listID,
		WordText:        wordText,
		DifficultyLevel: difficulty,
		Definition:      definition,
		Position:        position,
		CreatedAt:       time.Now(),
	}

	return word, nil
}

// GetListWords retrieves all words for a spelling list
func (r *ListRepository) GetListWords(listID int64) ([]models.Word, error) {
	query := `
		SELECT id, spelling_list_id, word_text, difficulty_level, audio_filename, definition, definition_audio_filename, position, created_at
		FROM words
		WHERE spelling_list_id = ?
		ORDER BY position ASC
	`
	rows, err := r.db.Query(query, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []models.Word
	for rows.Next() {
		var word models.Word
		var audioFilename, definition, definitionAudioFilename sql.NullString
		if err := rows.Scan(
			&word.ID,
			&word.SpellingListID,
			&word.WordText,
			&word.DifficultyLevel,
			&audioFilename,
			&definition,
			&definitionAudioFilename,
			&word.Position,
			&word.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		if audioFilename.Valid {
			word.AudioFilename = audioFilename.String
		}
		if definition.Valid {
			word.Definition = definition.String
		}
		if definitionAudioFilename.Valid {
			word.DefinitionAudioFilename = definitionAudioFilename.String
		}
		words = append(words, word)
	}

	return words, nil
}

// GetWordByID retrieves a word by ID
func (r *ListRepository) GetWordByID(wordID int64) (*models.Word, error) {
	query := `
		SELECT id, spelling_list_id, word_text, difficulty_level, audio_filename, position, created_at
		FROM words
		WHERE id = ?
	`
	word := &models.Word{}
	var audioFilename sql.NullString
	err := r.db.QueryRow(query, wordID).Scan(
		&word.ID,
		&word.SpellingListID,
		&word.WordText,
		&word.DifficultyLevel,
		&audioFilename,
		&word.Position,
		&word.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get word: %w", err)
	}

	if audioFilename.Valid {
		word.AudioFilename = audioFilename.String
	}

	return word, nil
}

// UpdateWord updates a word's text and difficulty
func (r *ListRepository) UpdateWord(wordID int64, wordText string, difficulty int) error {
	query := "UPDATE words SET word_text = ?, difficulty_level = ? WHERE id = ?"
	_, err := r.db.Exec(query, wordText, difficulty, wordID)
	if err != nil {
		return fmt.Errorf("failed to update word: %w", err)
	}
	return nil
}

// UpdateWordAudio updates a word's audio filename
func (r *ListRepository) UpdateWordAudio(wordID int64, audioFilename string) error {
	query := "UPDATE words SET audio_filename = ? WHERE id = ?"
	_, err := r.db.Exec(query, audioFilename, wordID)
	if err != nil {
		return fmt.Errorf("failed to update word audio: %w", err)
	}
	return nil
}

// UpdateWordDefinitionAudio updates a word's definition audio filename
func (r *ListRepository) UpdateWordDefinitionAudio(wordID int64, definitionAudioFilename string) error {
	query := "UPDATE words SET definition_audio_filename = ? WHERE id = ?"
	_, err := r.db.Exec(query, definitionAudioFilename, wordID)
	if err != nil {
		return fmt.Errorf("failed to update word definition audio: %w", err)
	}
	return nil
}

// DeleteWord deletes a word from a list
func (r *ListRepository) DeleteWord(wordID int64) error {
	query := "DELETE FROM words WHERE id = ?"
	_, err := r.db.Exec(query, wordID)
	if err != nil {
		return fmt.Errorf("failed to delete word: %w", err)
	}
	return nil
}

// GetWordCount returns the number of words in a list
func (r *ListRepository) GetWordCount(listID int64) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM words WHERE spelling_list_id = ?"
	err := r.db.QueryRow(query, listID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count words: %w", err)
	}
	return count, nil
}

// AssignListToKid assigns a spelling list to a kid
func (r *ListRepository) AssignListToKid(listID, kidID, assignedBy int64) error {
	query := "INSERT INTO list_assignments (spelling_list_id, kid_id, assigned_by) VALUES (?, ?, ?)"
	_, err := r.db.Exec(query, listID, kidID, assignedBy)
	if err != nil {
		return fmt.Errorf("failed to assign list: %w", err)
	}
	return nil
}

// UnassignListFromKid removes a list assignment
func (r *ListRepository) UnassignListFromKid(listID, kidID int64) error {
	query := "DELETE FROM list_assignments WHERE spelling_list_id = ? AND kid_id = ?"
	_, err := r.db.Exec(query, listID, kidID)
	if err != nil {
		return fmt.Errorf("failed to unassign list: %w", err)
	}
	return nil
}

// GetKidAssignedLists retrieves all lists assigned to a kid
func (r *ListRepository) GetKidAssignedLists(kidID int64) ([]models.SpellingList, error) {
	query := `
		SELECT sl.id, sl.family_id, sl.name, sl.description, sl.created_by, sl.created_at, sl.updated_at
		FROM spelling_lists sl
		INNER JOIN list_assignments la ON sl.id = la.spelling_list_id
		WHERE la.kid_id = ?
		ORDER BY la.assigned_at DESC
	`
	rows, err := r.db.Query(query, kidID)
	if err != nil {
		return nil, fmt.Errorf("failed to query assigned lists: %w", err)
	}
	defer rows.Close()

	var lists []models.SpellingList
	for rows.Next() {
		var list models.SpellingList
		if err := rows.Scan(
			&list.ID,
			&list.FamilyID,
			&list.Name,
			&list.Description,
			&list.CreatedBy,
			&list.CreatedAt,
			&list.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, nil
}

// GetListAssignedKids retrieves all kids assigned to a list
func (r *ListRepository) GetListAssignedKids(listID int64) ([]models.Kid, error) {
	query := `
		SELECT k.id, k.family_id, k.name, k.avatar_color, k.created_at, k.updated_at
		FROM kids k
		INNER JOIN list_assignments la ON k.id = la.kid_id
		WHERE la.spelling_list_id = ?
		ORDER BY k.name ASC
	`
	rows, err := r.db.Query(query, listID)
	if err != nil {
		return nil, fmt.Errorf("failed to query assigned kids: %w", err)
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

// GetFamilyListsWithAssignmentCounts retrieves all spelling lists for a family with assignment counts
func (r *ListRepository) GetFamilyListsWithAssignmentCounts(familyID int64) ([]models.ListSummary, error) {
	query := `
		SELECT 
			sl.id, sl.family_id, sl.name, sl.description, sl.created_by, sl.created_at, sl.updated_at,
			COUNT(DISTINCT la.kid_id) as assigned_kid_count
		FROM spelling_lists sl
		LEFT JOIN list_assignments la ON sl.id = la.spelling_list_id
		WHERE sl.family_id = ?
		GROUP BY sl.id
		ORDER BY sl.created_at DESC
	`
	rows, err := r.db.Query(query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lists with assignments: %w", err)
	}
	defer rows.Close()

	var lists []models.ListSummary
	for rows.Next() {
		var list models.ListSummary
		if err := rows.Scan(
			&list.ID,
			&list.FamilyID,
			&list.Name,
			&list.Description,
			&list.CreatedBy,
			&list.CreatedAt,
			&list.UpdatedAt,
			&list.AssignedKidCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan list: %w", err)
		}
		lists = append(lists, list)
	}

	return lists, nil
}

// IsListAssignedToKid checks if a list is assigned to a kid
func (r *ListRepository) IsListAssignedToKid(listID, kidID int64) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM list_assignments WHERE spelling_list_id = ? AND kid_id = ?"
	err := r.db.QueryRow(query, listID, kidID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check assignment: %w", err)
	}
	return count > 0, nil
}
