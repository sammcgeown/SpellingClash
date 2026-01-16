-- Missing Letter Mayhem Game Tables

-- Missing Letter Sessions
CREATE TABLE IF NOT EXISTS missing_letter_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kid_id INTEGER NOT NULL,
    spelling_list_id INTEGER NOT NULL,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    total_games INTEGER NOT NULL,
    games_won INTEGER DEFAULT 0,
    total_points INTEGER DEFAULT 0,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_missing_letter_sessions_kid ON missing_letter_sessions(kid_id);
CREATE INDEX IF NOT EXISTS idx_missing_letter_sessions_list ON missing_letter_sessions(spelling_list_id);

-- Missing Letter Games
CREATE TABLE IF NOT EXISTS missing_letter_games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id INTEGER NOT NULL,
    kid_id INTEGER NOT NULL,
    word_id INTEGER NOT NULL,
    word TEXT NOT NULL,
    missing_indices TEXT NOT NULL DEFAULT '[]',
    guessed_letters TEXT NOT NULL DEFAULT '[]',
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    is_won INTEGER DEFAULT 0,
    is_lost INTEGER DEFAULT 0,
    points_earned INTEGER DEFAULT 0,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (session_id) REFERENCES missing_letter_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_missing_letter_games_session ON missing_letter_games(session_id);
CREATE INDEX IF NOT EXISTS idx_missing_letter_games_kid ON missing_letter_games(kid_id);

-- Missing Letter State (for persisting game progress)
CREATE TABLE IF NOT EXISTS missing_letter_state (
    kid_id INTEGER PRIMARY KEY,
    session_id INTEGER NOT NULL,
    current_word_idx INTEGER DEFAULT 0,
    words_json TEXT NOT NULL,
    points_so_far INTEGER DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES missing_letter_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_missing_letter_state_session ON missing_letter_state(session_id);
