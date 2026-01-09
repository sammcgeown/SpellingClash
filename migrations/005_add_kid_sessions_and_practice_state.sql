-- Add kid sessions table for proper session management
CREATE TABLE IF NOT EXISTS kid_sessions (
    id TEXT PRIMARY KEY, -- UUID
    kid_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_kid_sessions_kid ON kid_sessions(kid_id);
CREATE INDEX IF NOT EXISTS idx_kid_sessions_expires ON kid_sessions(expires_at);

-- Add practice state table to persist practice sessions in progress
CREATE TABLE IF NOT EXISTS practice_state (
    kid_id INTEGER PRIMARY KEY,
    session_id INTEGER NOT NULL,
    current_index INTEGER NOT NULL DEFAULT 0,
    correct_count INTEGER NOT NULL DEFAULT 0,
    total_points INTEGER NOT NULL DEFAULT 0,
    start_time DATETIME NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE
);

-- Track when each word was presented to calculate time accurately
CREATE TABLE IF NOT EXISTS practice_word_timing (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kid_id INTEGER NOT NULL,
    session_id INTEGER NOT NULL,
    word_index INTEGER NOT NULL,
    started_at DATETIME NOT NULL,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE,
    UNIQUE(kid_id, session_id, word_index)
);

CREATE INDEX IF NOT EXISTS idx_practice_word_timing_kid_session ON practice_word_timing(kid_id, session_id);
