-- Missing Letter Mayhem Game Tables

-- Missing Letter Sessions
CREATE TABLE IF NOT EXISTS missing_letter_sessions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    kid_id BIGINT NOT NULL,
    spelling_list_id BIGINT NOT NULL,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    total_games INTEGER NOT NULL,
    games_won INTEGER DEFAULT 0,
    total_points INTEGER DEFAULT 0,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
);

CREATE INDEX idx_missing_letter_sessions_kid ON missing_letter_sessions(kid_id);
CREATE INDEX idx_missing_letter_sessions_list ON missing_letter_sessions(spelling_list_id);

-- Missing Letter Games
CREATE TABLE IF NOT EXISTS missing_letter_games (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    session_id BIGINT NOT NULL,
    kid_id BIGINT NOT NULL,
    word_id BIGINT NOT NULL,
    word TEXT NOT NULL,
    missing_indices TEXT NOT NULL,
    guessed_letters TEXT NOT NULL,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    is_won BOOLEAN DEFAULT FALSE,
    is_lost BOOLEAN DEFAULT FALSE,
    points_earned INTEGER DEFAULT 0,
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    FOREIGN KEY (session_id) REFERENCES missing_letter_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);

CREATE INDEX idx_missing_letter_games_session ON missing_letter_games(session_id);
CREATE INDEX idx_missing_letter_games_kid ON missing_letter_games(kid_id);

-- Missing Letter State (for persisting game progress)
CREATE TABLE IF NOT EXISTS missing_letter_state (
    kid_id BIGINT PRIMARY KEY,
    session_id BIGINT NOT NULL,
    current_word_idx INTEGER DEFAULT 0,
    words_json TEXT NOT NULL,
    points_so_far INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES missing_letter_sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_missing_letter_state_session ON missing_letter_state(session_id);
