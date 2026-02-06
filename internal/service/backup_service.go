package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"spellingclash/internal/database"
	"time"
)

// BackupData represents the complete database backup structure
type BackupData struct {
	Version      string          `json:"version"`
	ExportedAt   time.Time       `json:"exported_at"`
	DatabaseType string          `json:"database_type"`
	Users        []UserBackup    `json:"users"`
	Families     []FamilyBackup  `json:"families"`
	Kids         []KidBackup     `json:"kids"`
	Lists        []ListBackup    `json:"lists"`
	Words        []WordBackup    `json:"words"`
	Practices    []PracticeBackup `json:"practices"`
}

// UserBackup represents a user record for backup
type UserBackup struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"password_hash"`
	Name          string    `json:"name"`
	OAuthProvider string    `json:"oauth_provider"`
	OAuthSubject  string    `json:"oauth_subject"`
	IsAdmin       bool      `json:"is_admin"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FamilyBackup represents a family record for backup
type FamilyBackup struct {
	FamilyCode string    `json:"family_code"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Members    []FamilyMemberBackup `json:"members"`
}

// FamilyMemberBackup represents a family member record
type FamilyMemberBackup struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

// KidBackup represents a kid record for backup
type KidBackup struct {
	ID          int64     `json:"id"`
	FamilyCode  string    `json:"family_code"`
	Name        string    `json:"name"`
	Username    string    `json:"username"`
	Password    string    `json:"password"`
	AvatarColor string    `json:"avatar_color"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListBackup represents a spelling list for backup
type ListBackup struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	FamilyCode  *string   `json:"family_code"`
	IsPublic    bool      `json:"is_public"`
	CreatedBy   *int64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Assignments []int64   `json:"assigned_kids"`
}

// WordBackup represents a word for backup
type WordBackup struct {
	ID                      int64  `json:"id"`
	SpellingListID          int64  `json:"spelling_list_id"`
	WordText                string `json:"word_text"`
	DifficultyLevel         int    `json:"difficulty_level"`
	AudioFilename           string `json:"audio_filename"`
	Definition              string `json:"definition"`
	DefinitionAudioFilename string `json:"definition_audio_filename"`
	Position                int    `json:"position"`
	CreatedAt               time.Time `json:"created_at"`
}

// PracticeBackup represents a practice session for backup
type PracticeBackup struct {
	ID             int64     `json:"id"`
	KidID          int64     `json:"kid_id"`
	SpellingListID int64     `json:"spelling_list_id"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	TotalWords     int       `json:"total_words"`
	CorrectWords   int       `json:"correct_words"`
	PointsEarned   int       `json:"points_earned"`
}

// PracticeResultBackup represents a practice result for backup
type PracticeResultBackup struct {
	WordID  int64  `json:"word_id"`
	Correct bool   `json:"correct"`
	Answer  string `json:"answer"`
}

// BackupService handles database backup and restore operations
type BackupService struct {
	db *database.DB
}

// NewBackupService creates a new backup service
func NewBackupService(db *database.DB) *BackupService {
	return &BackupService{db: db}
}

// GetDB returns the database connection for direct queries
func (s *BackupService) GetDB() *database.DB {
	return s.db
}

// Export creates a complete backup of the database to a file
func (s *BackupService) Export(outputPath string) error {
	log.Println("Starting database export...")
	
	backup := &BackupData{
		Version:      "1.0",
		ExportedAt:   time.Now(),
		DatabaseType: "universal",
	}

	// Export users
	if err := s.exportUsers(backup); err != nil {
		return fmt.Errorf("failed to export users: %w", err)
	}

	// Export families
	if err := s.exportFamilies(backup); err != nil {
		return fmt.Errorf("failed to export families: %w", err)
	}

	// Export kids
	if err := s.exportKids(backup); err != nil {
		return fmt.Errorf("failed to export kids: %w", err)
	}

	// Export lists
	if err := s.exportLists(backup); err != nil {
		return fmt.Errorf("failed to export lists: %w", err)
	}

	// Export words
	if err := s.exportWords(backup); err != nil {
		return fmt.Errorf("failed to export words: %w", err)
	}

	// Export practice sessions
	if err := s.exportPractices(backup); err != nil {
		return fmt.Errorf("failed to export practices: %w", err)
	}

	// Write to file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(backup); err != nil {
		return fmt.Errorf("failed to encode backup: %w", err)
	}

	log.Printf("Database exported successfully to %s", outputPath)
	log.Printf("Exported: %d users, %d families, %d kids, %d lists, %d words, %d practices",
		len(backup.Users), len(backup.Families), len(backup.Kids), 
		len(backup.Lists), len(backup.Words), len(backup.Practices))

	return nil
}

// Import restores a database from a backup file
func (s *BackupService) Import(inputPath string) error {
	log.Printf("Starting database import from %s...", inputPath)

	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	var backup BackupData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&backup); err != nil {
		return fmt.Errorf("failed to decode backup: %w", err)
	}

	log.Printf("Backup version: %s, exported at: %s", backup.Version, backup.ExportedAt)

	// Import in order of dependencies
	if err := s.importUsers(backup.Users); err != nil {
		return fmt.Errorf("failed to import users: %w", err)
	}

	if err := s.importFamilies(backup.Families); err != nil {
		return fmt.Errorf("failed to import families: %w", err)
	}

	if err := s.importKids(backup.Kids); err != nil {
		return fmt.Errorf("failed to import kids: %w", err)
	}

	if err := s.importLists(backup.Lists); err != nil {
		return fmt.Errorf("failed to import lists: %w", err)
	}

	if err := s.importWords(backup.Words); err != nil {
		return fmt.Errorf("failed to import words: %w", err)
	}

	if err := s.importPractices(backup.Practices); err != nil {
		return fmt.Errorf("failed to import practices: %w", err)
	}

	log.Println("Database import completed successfully")
	return nil
}

// ImportFromReader restores a database from a backup reader (for file uploads)
func (s *BackupService) ImportFromReader(reader io.Reader) error {
	log.Println("Starting database import from reader...")

	var backup BackupData
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&backup); err != nil {
		return fmt.Errorf("failed to decode backup: %w", err)
	}

	log.Printf("Backup version: %s, exported at: %s", backup.Version, backup.ExportedAt)

	// Import in order of dependencies
	if err := s.importUsers(backup.Users); err != nil {
		return fmt.Errorf("failed to import users: %w", err)
	}

	if err := s.importFamilies(backup.Families); err != nil {
		return fmt.Errorf("failed to import families: %w", err)
	}

	if err := s.importKids(backup.Kids); err != nil {
		return fmt.Errorf("failed to import kids: %w", err)
	}

	if err := s.importLists(backup.Lists); err != nil {
		return fmt.Errorf("failed to import lists: %w", err)
	}

	if err := s.importWords(backup.Words); err != nil {
		return fmt.Errorf("failed to import words: %w", err)
	}

	if err := s.importPractices(backup.Practices); err != nil {
		return fmt.Errorf("failed to import practices: %w", err)
	}

	log.Println("Database import completed successfully")
	return nil
}

func (s *BackupService) exportUsers(backup *BackupData) error {
	query := "SELECT id, email, password_hash, name, COALESCE(oauth_provider, ''), COALESCE(oauth_subject, ''), is_admin, created_at, updated_at FROM users ORDER BY id"
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var u UserBackup
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.OAuthProvider, &u.OAuthSubject, &u.IsAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return err
		}
		backup.Users = append(backup.Users, u)
	}
	return rows.Err()
}

func (s *BackupService) exportFamilies(backup *BackupData) error {
	query := "SELECT family_code, created_at, updated_at FROM families ORDER BY family_code"
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var f FamilyBackup
		if err := rows.Scan(&f.FamilyCode, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return err
		}

		// Get family members
		memberQuery := "SELECT user_id, role FROM family_members WHERE family_code = ? ORDER BY user_id"
		memberRows, err := s.db.Query(memberQuery, f.FamilyCode)
		if err != nil {
			return err
		}

		for memberRows.Next() {
			var m FamilyMemberBackup
			if err := memberRows.Scan(&m.UserID, &m.Role); err != nil {
				memberRows.Close()
				return err
			}
			f.Members = append(f.Members, m)
		}
		memberRows.Close()

		backup.Families = append(backup.Families, f)
	}
	return rows.Err()
}

func (s *BackupService) exportKids(backup *BackupData) error {
	query := "SELECT id, family_code, name, username, COALESCE(password, ''), COALESCE(avatar_color, '#4A90E2'), created_at, updated_at FROM kids ORDER BY id"
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var k KidBackup
		if err := rows.Scan(&k.ID, &k.FamilyCode, &k.Name, &k.Username, &k.Password, &k.AvatarColor, &k.CreatedAt, &k.UpdatedAt); err != nil {
			return err
		}
		backup.Kids = append(backup.Kids, k)
	}
	return rows.Err()
}

func (s *BackupService) exportLists(backup *BackupData) error {
	query := "SELECT id, name, description, family_code, is_public, created_by, created_at, updated_at FROM spelling_lists WHERE is_public = 0 ORDER BY id"
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var l ListBackup
		var familyCode sql.NullString
		var createdBy sql.NullInt64
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &familyCode, &l.IsPublic, &createdBy, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return err
		}
		if familyCode.Valid {
			l.FamilyCode = &familyCode.String
		}
		if createdBy.Valid {
			l.CreatedBy = &createdBy.Int64
		}

		// Get assigned kids
		assignQuery := "SELECT kid_id FROM list_assignments WHERE spelling_list_id = ? ORDER BY kid_id"
		assignRows, err := s.db.Query(assignQuery, l.ID)
		if err != nil {
			return err
		}

		for assignRows.Next() {
			var kidID int64
			if err := assignRows.Scan(&kidID); err != nil {
				assignRows.Close()
				return err
			}
			l.Assignments = append(l.Assignments, kidID)
		}
		assignRows.Close()

		backup.Lists = append(backup.Lists, l)
	}
	return rows.Err()
}

func (s *BackupService) exportWords(backup *BackupData) error {
	query := "SELECT w.id, w.spelling_list_id, w.word_text, COALESCE(w.difficulty_level, 1), COALESCE(w.audio_filename, ''), COALESCE(w.definition, ''), COALESCE(w.definition_audio_filename, ''), w.position, w.created_at FROM words w JOIN spelling_lists sl ON w.spelling_list_id = sl.id WHERE sl.is_public = 0 ORDER BY w.id"
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var w WordBackup
		if err := rows.Scan(&w.ID, &w.SpellingListID, &w.WordText, &w.DifficultyLevel, &w.AudioFilename, &w.Definition, &w.DefinitionAudioFilename, &w.Position, &w.CreatedAt); err != nil {
			return err
		}
		backup.Words = append(backup.Words, w)
	}
	return rows.Err()
}

func (s *BackupService) exportPractices(backup *BackupData) error {
	query := "SELECT ps.id, ps.kid_id, ps.spelling_list_id, ps.started_at, ps.completed_at, ps.total_words, ps.correct_words, ps.points_earned FROM practice_sessions ps JOIN spelling_lists sl ON ps.spelling_list_id = sl.id WHERE sl.is_public = 0 ORDER BY ps.id"
	rows, err := s.db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var p PracticeBackup
		var completedAt sql.NullTime
		if err := rows.Scan(&p.ID, &p.KidID, &p.SpellingListID, &p.StartedAt, &completedAt, &p.TotalWords, &p.CorrectWords, &p.PointsEarned); err != nil {
			return err
		}
		if completedAt.Valid {
			p.CompletedAt = &completedAt.Time
		}
		backup.Practices = append(backup.Practices, p)
	}
	return rows.Err()
}

func (s *BackupService) importUsers(users []UserBackup) error {
	log.Printf("Importing %d users...", len(users))
	for _, u := range users {
		query := "INSERT INTO users (id, email, password_hash, name, oauth_provider, oauth_subject, is_admin, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
		_, err := s.db.Exec(query, u.ID, u.Email, u.PasswordHash, u.Name, nullIfEmpty(u.OAuthProvider), nullIfEmpty(u.OAuthSubject), u.IsAdmin, u.CreatedAt, u.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to import user %d: %w", u.ID, err)
		}
	}
	return nil
}

func (s *BackupService) importFamilies(families []FamilyBackup) error {
	log.Printf("Importing %d families...", len(families))
	for _, f := range families {
		query := "INSERT INTO families (family_code, created_at, updated_at) VALUES (?, ?, ?)"
		_, err := s.db.Exec(query, f.FamilyCode, f.CreatedAt, f.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to import family %s: %w", f.FamilyCode, err)
		}

		// Import family members
		for _, m := range f.Members {
			memberQuery := "INSERT INTO family_members (family_code, user_id, role) VALUES (?, ?, ?)"
			_, err := s.db.Exec(memberQuery, f.FamilyCode, m.UserID, m.Role)
			if err != nil {
				return fmt.Errorf("failed to import family member %d for family %s: %w", m.UserID, f.FamilyCode, err)
			}
		}
	}
	return nil
}

func (s *BackupService) importKids(kids []KidBackup) error {
	log.Printf("Importing %d kids...", len(kids))
	for _, k := range kids {
		query := "INSERT INTO kids (id, family_code, name, username, password, avatar_color, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
		_, err := s.db.Exec(query, k.ID, k.FamilyCode, k.Name, k.Username, nullIfEmpty(k.Password), k.AvatarColor, k.CreatedAt, k.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to import kid %d: %w", k.ID, err)
		}
	}
	return nil
}

func (s *BackupService) importLists(lists []ListBackup) error {
	log.Printf("Importing %d lists...", len(lists))
	for _, l := range lists {
		var familyCode interface{} = nil
		if l.FamilyCode != nil {
			familyCode = *l.FamilyCode
		}
		var createdBy interface{} = nil
		if l.CreatedBy != nil {
			createdBy = *l.CreatedBy
		}
		query := "INSERT INTO spelling_lists (id, name, description, family_code, is_public, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
		_, err := s.db.Exec(query, l.ID, l.Name, l.Description, familyCode, l.IsPublic, createdBy, l.CreatedAt, l.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to import list %d: %w", l.ID, err)
		}

		// Import assignments
		for _, kidID := range l.Assignments {
			assignQuery := "INSERT INTO list_assignments (spelling_list_id, kid_id, assigned_by) VALUES (?, ?, ?)"
			assignedBy := 1 // Default to admin user
			if l.CreatedBy != nil {
				assignedBy = int(*l.CreatedBy)
			}
			_, err := s.db.Exec(assignQuery, l.ID, kidID, assignedBy)
			if err != nil {
				return fmt.Errorf("failed to import assignment for list %d, kid %d: %w", l.ID, kidID, err)
			}
		}
	}
	return nil
}

func (s *BackupService) importWords(words []WordBackup) error {
	log.Printf("Importing %d words...", len(words))
	for _, w := range words {
		query := "INSERT INTO words (id, spelling_list_id, word_text, difficulty_level, audio_filename, definition, definition_audio_filename, position, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
		_, err := s.db.Exec(query, w.ID, w.SpellingListID, w.WordText, w.DifficultyLevel, nullIfEmpty(w.AudioFilename), nullIfEmpty(w.Definition), nullIfEmpty(w.DefinitionAudioFilename), w.Position, w.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to import word %d: %w", w.ID, err)
		}
	}
	return nil
}

func (s *BackupService) importPractices(practices []PracticeBackup) error {
	log.Printf("Importing %d practice sessions...", len(practices))
	for _, p := range practices {
		var completedAt interface{} = nil
		if p.CompletedAt != nil {
			completedAt = *p.CompletedAt
		}
		query := "INSERT INTO practice_sessions (id, kid_id, spelling_list_id, started_at, completed_at, total_words, correct_words, points_earned) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
		_, err := s.db.Exec(query, p.ID, p.KidID, p.SpellingListID, p.StartedAt, completedAt, p.TotalWords, p.CorrectWords, p.PointsEarned)
		if err != nil {
			return fmt.Errorf("failed to import practice %d: %w", p.ID, err)
		}
	}
	return nil
}

// ExportToWriter exports the database to an io.Writer (useful for HTTP responses)
func (s *BackupService) ExportToWriter(w io.Writer) error {
	backup := &BackupData{
		Version:      "1.0",
		ExportedAt:   time.Now(),
		DatabaseType: "universal",
	}

	if err := s.exportUsers(backup); err != nil {
		return fmt.Errorf("failed to export users: %w", err)
	}
	if err := s.exportFamilies(backup); err != nil {
		return fmt.Errorf("failed to export families: %w", err)
	}
	if err := s.exportKids(backup); err != nil {
		return fmt.Errorf("failed to export kids: %w", err)
	}
	if err := s.exportLists(backup); err != nil {
		return fmt.Errorf("failed to export lists: %w", err)
	}
	if err := s.exportWords(backup); err != nil {
		return fmt.Errorf("failed to export words: %w", err)
	}
	if err := s.exportPractices(backup); err != nil {
		return fmt.Errorf("failed to export practices: %w", err)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(backup)
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
