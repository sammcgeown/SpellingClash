package repository

import (
	"database/sql"
	"fmt"
	"spellingclash/internal/database"
	"spellingclash/internal/models"
	"time"
)

// UserRepository handles database operations for users and sessions
type UserRepository struct {
	db *database.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user into the database
func (r *UserRepository) CreateUser(email, passwordHash, name string) (*models.User, error) {
	// Check if this is the first user
	var userCount int
	countQuery := "SELECT COUNT(*) FROM users"
	err := r.db.QueryRow(countQuery).Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// First user becomes admin
	isAdmin := userCount == 0

	query := `
		INSERT INTO users (email, password_hash, name, is_admin)
		VALUES (?, ?, ?, ?)
	`
	id, err := r.db.ExecReturningID(query, email, passwordHash, name, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user := &models.User{
		ID:           id,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		OAuthProvider: "",
		OAuthSubject:  "",
		IsAdmin:      isAdmin,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email address
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, COALESCE(oauth_provider, ''), COALESCE(oauth_subject, ''), is_admin, created_at, updated_at
		FROM users
		WHERE email = ?
	`
	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.OAuthProvider,
		&user.OAuthSubject,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(id int64) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, COALESCE(oauth_provider, ''), COALESCE(oauth_subject, ''), is_admin, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.OAuthProvider,
		&user.OAuthSubject,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// CreateSession creates a new session for a user
func (r *UserRepository) CreateSession(sessionID string, userID int64, expiresAt time.Time) (*models.Session, error) {
	query := `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`
	_, err := r.db.Exec(query, sessionID, userID, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	session := &models.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (r *UserRepository) GetSession(sessionID string) (*models.Session, error) {
	query := `
		SELECT id, user_id, expires_at, created_at
		FROM sessions
		WHERE id = ?
	`
	session := &models.Session{}
	err := r.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// DeleteSession removes a session from the database
func (r *UserRepository) DeleteSession(sessionID string) error {
	query := "DELETE FROM sessions WHERE id = ?"
	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteExpiredSessions removes all expired sessions
func (r *UserRepository) DeleteExpiredSessions() error {
	query := "DELETE FROM sessions WHERE expires_at < ?"
	_, err := r.db.Exec(query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}

// GetAllUsers retrieves all users
func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	query := `
		SELECT id, email, password_hash, name, COALESCE(oauth_provider, ''), COALESCE(oauth_subject, ''), is_admin, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Name,
			&user.OAuthProvider,
			&user.OAuthSubject,
			&user.IsAdmin,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateUser updates a user's information
func (r *UserRepository) UpdateUser(id int64, email, name string, isAdmin bool) error {
	query := `
		UPDATE users
		SET email = ?, name = ?, is_admin = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Exec(query, email, name, isAdmin, id)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// DeleteUser deletes a user and all associated data
func (r *UserRepository) DeleteUser(id int64) error {
	query := "DELETE FROM users WHERE id = ?"
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// GetUserByOAuth retrieves a user by OAuth provider and subject
func (r *UserRepository) GetUserByOAuth(provider, subject string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, name, COALESCE(oauth_provider, ''), COALESCE(oauth_subject, ''), is_admin, created_at, updated_at
		FROM users
		WHERE oauth_provider = ? AND oauth_subject = ?
	`
	user := &models.User{}
	err := r.db.QueryRow(query, provider, subject).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.OAuthProvider,
		&user.OAuthSubject,
		&user.IsAdmin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by oauth: %w", err)
	}

	return user, nil
}

// LinkOAuthProvider links an existing user to an OAuth provider
func (r *UserRepository) LinkOAuthProvider(userID int64, provider, subject string) error {
	query := `
		UPDATE users
		SET oauth_provider = ?, oauth_subject = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
		AND (oauth_provider IS NULL OR oauth_provider = '')
	`
	result, err := r.db.Exec(query, provider, subject, userID)
	if err != nil {
		return fmt.Errorf("failed to link oauth provider: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to read link result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("oauth provider already linked")
	}

	return nil
}
