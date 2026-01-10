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

// SavePracticeState saves the current practice state for a kid
func (r *PracticeRepository) SavePracticeState(kidID, sessionID int64, currentIndex, correctCount, totalPoints int, startTime time.Time, wordOrder string) error {
	query := `
		INSERT OR REPLACE INTO practice_state 
		(kid_id, session_id, current_index, correct_count, total_points, start_time, word_order, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := r.db.Exec(query, kidID, sessionID, currentIndex, correctCount, totalPoints, startTime, wordOrder)
	return err
}

// GetPracticeState retrieves the current practice state for a kid
func (r *PracticeRepository) GetPracticeState(kidID int64) (*models.PracticeState, error) {
	query := `
		SELECT kid_id, session_id, current_index, correct_count, total_points, start_time, updated_at, COALESCE(word_order, '')
		FROM practice_state
		WHERE kid_id = ?
	`
	
	state := &models.PracticeState{}
	err := r.db.QueryRow(query, kidID).Scan(
		&state.KidID,
		&state.SessionID,
		&state.CurrentIndex,
		&state.CorrectCount,
		&state.TotalPoints,
		&state.StartTime,
		&state.UpdatedAt,
		&state.WordOrder,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return state, nil
}

// DeletePracticeState removes the practice state for a kid
func (r *PracticeRepository) DeletePracticeState(kidID int64) error {
	query := "DELETE FROM practice_state WHERE kid_id = ?"
	_, err := r.db.Exec(query, kidID)
	return err
}

// SaveWordTiming saves when a word was presented to the kid
func (r *PracticeRepository) SaveWordTiming(kidID, sessionID int64, wordIndex int, startedAt time.Time) error {
	query := `
		INSERT OR REPLACE INTO practice_word_timing 
		(kid_id, session_id, word_index, started_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, kidID, sessionID, wordIndex, startedAt)
	return err
}

// GetWordTiming retrieves when a word was presented
func (r *PracticeRepository) GetWordTiming(kidID, sessionID int64, wordIndex int) (time.Time, error) {
	query := `
		SELECT started_at
		FROM practice_word_timing
		WHERE kid_id = ? AND session_id = ? AND word_index = ?
	`
	
	var startedAt time.Time
	err := r.db.QueryRow(query, kidID, sessionID, wordIndex).Scan(&startedAt)
	if err == sql.ErrNoRows {
		return time.Now(), nil // Return current time if not found
	}
	return startedAt, err
}

// DeleteWordTimings removes all word timings for a session
func (r *PracticeRepository) DeleteWordTimings(kidID, sessionID int64) error {
	query := "DELETE FROM practice_word_timing WHERE kid_id = ? AND session_id = ?"
	_, err := r.db.Exec(query, kidID, sessionID)
	return err
}

// WordPerformance represents a word's performance statistics for a kid
type WordPerformance struct {
	WordID         int64
	TotalAttempts  int
	CorrectAttempts int
	SuccessRate    float64
}

// GetWordPerformanceForKid gets performance statistics for all words for a specific kid
func (r *PracticeRepository) GetWordPerformanceForKid(kidID int64, wordIDs []int64) (map[int64]*WordPerformance, error) {
	if len(wordIDs) == 0 {
		return make(map[int64]*WordPerformance), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]interface{}, len(wordIDs)+1)
	placeholders[0] = kidID
	for i, id := range wordIDs {
		placeholders[i+1] = id
	}

	query := `
		SELECT 
			wa.word_id,
			COUNT(*) as total_attempts,
			SUM(CASE WHEN wa.is_correct = 1 THEN 1 ELSE 0 END) as correct_attempts
		FROM word_attempts wa
		JOIN practice_sessions ps ON wa.practice_session_id = ps.id
		WHERE ps.kid_id = ?
		AND wa.word_id IN (` + generatePlaceholders(len(wordIDs)) + `)
		GROUP BY wa.word_id
	`

	rows, err := r.db.Query(query, placeholders...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	performance := make(map[int64]*WordPerformance)
	for rows.Next() {
		var wordID int64
		var totalAttempts, correctAttempts int
		if err := rows.Scan(&wordID, &totalAttempts, &correctAttempts); err != nil {
			return nil, err
		}
		
		successRate := 0.0
		if totalAttempts > 0 {
			successRate = float64(correctAttempts) / float64(totalAttempts)
		}

		performance[wordID] = &WordPerformance{
			WordID:         wordID,
			TotalAttempts:  totalAttempts,
			CorrectAttempts: correctAttempts,
			SuccessRate:    successRate,
		}
	}

	return performance, nil
}

// generatePlaceholders generates SQL placeholders for IN clause
func generatePlaceholders(count int) string {
	if count == 0 {
		return ""
	}
	placeholders := make([]byte, count*2-1)
	for i := 0; i < count; i++ {
		if i > 0 {
			placeholders[i*2-1] = ','
		}
		placeholders[i*2] = '?'
	}
	return string(placeholders)
}

// StrugglingWord represents a word a kid is having trouble with
type StrugglingWord struct {
	WordID          int64
	WordText        string
	TotalAttempts   int
	CorrectAttempts int
	SuccessRate     float64
	LastAttempted   time.Time
}

// GetStrugglingWordsForKid gets words with low success rates for a kid
// threshold is the success rate below which a word is considered struggling (e.g., 0.6 for 60%)
// minAttempts is the minimum number of attempts before considering a word
func (r *PracticeRepository) GetStrugglingWordsForKid(kidID int64, threshold float64, minAttempts int) ([]StrugglingWord, error) {
	query := `
		SELECT 
			wa.word_id,
			w.word_text,
			COUNT(*) as total_attempts,
			SUM(CASE WHEN wa.is_correct = 1 THEN 1 ELSE 0 END) as correct_attempts,
			MAX(ps.started_at) as last_attempted
		FROM word_attempts wa
		JOIN practice_sessions ps ON wa.practice_session_id = ps.id
		JOIN words w ON wa.word_id = w.id
		WHERE ps.kid_id = ?
		GROUP BY wa.word_id, w.word_text
		HAVING COUNT(*) >= ?
		ORDER BY 
			(CAST(SUM(CASE WHEN wa.is_correct = 1 THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*)) ASC,
			COUNT(*) DESC
	`

	rows, err := r.db.Query(query, kidID, minAttempts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strugglingWords []StrugglingWord
	for rows.Next() {
		var word StrugglingWord
		var totalAttempts, correctAttempts int
		
		if err := rows.Scan(&word.WordID, &word.WordText, &totalAttempts, &correctAttempts, &word.LastAttempted); err != nil {
			return nil, err
		}
		
		word.TotalAttempts = totalAttempts
		word.CorrectAttempts = correctAttempts
		
		if totalAttempts > 0 {
			word.SuccessRate = float64(correctAttempts) / float64(totalAttempts)
		}
		
		// Only include if below threshold
		if word.SuccessRate < threshold {
			strugglingWords = append(strugglingWords, word)
		}
	}

	return strugglingWords, nil
}

// GetKidStats gets overall statistics for a kid's practice sessions
func (r *PracticeRepository) GetKidStats(kidID int64) (*models.KidStats, error) {
	query := `
		SELECT 
			COUNT(DISTINCT ps.id) as total_sessions,
			COUNT(wa.id) as total_attempts,
			COALESCE(SUM(CASE WHEN wa.is_correct = 1 THEN 1 ELSE 0 END), 0) as total_correct,
			COALESCE(SUM(wa.points_earned), 0) as total_points,
			COUNT(DISTINCT wa.word_id) as unique_words_attempted
		FROM practice_sessions ps
		LEFT JOIN word_attempts wa ON ps.id = wa.practice_session_id
		WHERE ps.kid_id = ? AND ps.completed_at IS NOT NULL
	`

	stats := &models.KidStats{}
	var totalAttempts, totalCorrect int

	err := r.db.QueryRow(query, kidID).Scan(
		&stats.TotalSessions,
		&totalAttempts,
		&totalCorrect,
		&stats.TotalPoints,
		&stats.UniqueWordsAttempted,
	)
	if err != nil {
		return nil, err
	}

	stats.TotalWordsPracticed = totalAttempts
	stats.TotalCorrect = totalCorrect
	
	if totalAttempts > 0 {
		stats.OverallAccuracy = (float64(totalCorrect) / float64(totalAttempts)) * 100
	}

	return stats, nil
}
