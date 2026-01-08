package repository

import (
	"database/sql"
	"time"
	"wordclash/internal/models"
)

// PracticeRepository handles practice session database operations
type PracticeRepository struct {
	db *sql.DB
}

// NewPracticeRepository creates a new practice repository
func NewPracticeRepository(db *sql.DB) *PracticeRepository {
	return &PracticeRepository{db: db}
}

// CreateSession creates a new practice session
func (r *PracticeRepository) CreateSession(kidID, listID int64, totalWords int) (*models.PracticeSession, error) {
	query := `
		INSERT INTO practice_sessions (kid_id, spelling_list_id, total_words)
		VALUES (?, ?, ?)
	`

	result, err := r.db.Exec(query, kidID, listID, totalWords)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetSessionByID(id)
}

// GetSessionByID retrieves a practice session by ID
func (r *PracticeRepository) GetSessionByID(sessionID int64) (*models.PracticeSession, error) {
	query := `
		SELECT id, kid_id, spelling_list_id, started_at, completed_at,
		       total_words, correct_words, points_earned
		FROM practice_sessions
		WHERE id = ?
	`

	session := &models.PracticeSession{}
	var completedAt sql.NullTime

	err := r.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.KidID,
		&session.SpellingListID,
		&session.StartedAt,
		&completedAt,
		&session.TotalWords,
		&session.CorrectWords,
		&session.PointsEarned,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		session.CompletedAt = &completedAt.Time
	}

	return session, nil
}

// RecordAttempt records a word attempt
func (r *PracticeRepository) RecordAttempt(sessionID, wordID int64, attemptText string, isCorrect bool, timeTakenMs, pointsEarned int) (*models.WordAttempt, error) {
	query := `
		INSERT INTO word_attempts (practice_session_id, word_id, attempt_text, is_correct, time_taken_ms, points_earned)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(query, sessionID, wordID, attemptText, isCorrect, timeTakenMs, pointsEarned)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &models.WordAttempt{
		ID:                id,
		PracticeSessionID: sessionID,
		WordID:            wordID,
		AttemptText:       attemptText,
		IsCorrect:         isCorrect,
		TimeTakenMs:       timeTakenMs,
		PointsEarned:      pointsEarned,
		AttemptedAt:       time.Now(),
	}, nil
}

// CompleteSession marks a session as complete and updates totals
func (r *PracticeRepository) CompleteSession(sessionID int64, correctWords, totalPoints int) error {
	query := `
		UPDATE practice_sessions
		SET completed_at = ?, correct_words = ?, points_earned = ?
		WHERE id = ?
	`

	_, err := r.db.Exec(query, time.Now(), correctWords, totalPoints, sessionID)
	return err
}

// GetSessionAttempts retrieves all attempts for a session
func (r *PracticeRepository) GetSessionAttempts(sessionID int64) ([]models.WordAttempt, error) {
	query := `
		SELECT id, practice_session_id, word_id, attempt_text, is_correct,
		       time_taken_ms, points_earned, attempted_at
		FROM word_attempts
		WHERE practice_session_id = ?
		ORDER BY attempted_at ASC
	`

	rows, err := r.db.Query(query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attempts []models.WordAttempt
	for rows.Next() {
		var attempt models.WordAttempt
		err := rows.Scan(
			&attempt.ID,
			&attempt.PracticeSessionID,
			&attempt.WordID,
			&attempt.AttemptText,
			&attempt.IsCorrect,
			&attempt.TimeTakenMs,
			&attempt.PointsEarned,
			&attempt.AttemptedAt,
		)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, attempt)
	}

	return attempts, rows.Err()
}

// GetKidSessions retrieves all sessions for a kid
func (r *PracticeRepository) GetKidSessions(kidID int64, limit int) ([]models.PracticeSession, error) {
	query := `
		SELECT id, kid_id, spelling_list_id, started_at, completed_at,
		       total_words, correct_words, points_earned
		FROM practice_sessions
		WHERE kid_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, kidID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.PracticeSession
	for rows.Next() {
		var session models.PracticeSession
		var completedAt sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.KidID,
			&session.SpellingListID,
			&session.StartedAt,
			&completedAt,
			&session.TotalWords,
			&session.CorrectWords,
			&session.PointsEarned,
		)
		if err != nil {
			return nil, err
		}

		if completedAt.Valid {
			session.CompletedAt = &completedAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// GetKidTotalPoints calculates total points earned by a kid
func (r *PracticeRepository) GetKidTotalPoints(kidID int64) (int, error) {
	query := `
		SELECT COALESCE(SUM(points_earned), 0)
		FROM practice_sessions
		WHERE kid_id = ? AND completed_at IS NOT NULL
	`

	var totalPoints int
	err := r.db.QueryRow(query, kidID).Scan(&totalPoints)
	return totalPoints, err
}
