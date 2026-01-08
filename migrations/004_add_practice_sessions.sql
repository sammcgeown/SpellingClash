-- Practice sessions table
CREATE TABLE IF NOT EXISTS practice_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kid_id INTEGER NOT NULL,
    spelling_list_id INTEGER NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    total_words INTEGER NOT NULL DEFAULT 0,
    correct_words INTEGER NOT NULL DEFAULT 0,
    points_earned INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
);

-- Word attempts table
CREATE TABLE IF NOT EXISTS word_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    practice_session_id INTEGER NOT NULL,
    word_id INTEGER NOT NULL,
    attempt_text TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT 0,
    time_taken_ms INTEGER NOT NULL,
    points_earned INTEGER NOT NULL DEFAULT 0,
    attempted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (practice_session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_practice_sessions_kid ON practice_sessions(kid_id);
CREATE INDEX IF NOT EXISTS idx_practice_sessions_list ON practice_sessions(spelling_list_id);
CREATE INDEX IF NOT EXISTS idx_word_attempts_session ON word_attempts(practice_session_id);
CREATE INDEX IF NOT EXISTS idx_word_attempts_word ON word_attempts(word_id);
