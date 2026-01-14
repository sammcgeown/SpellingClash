-- Hangman game tables (PostgreSQL)

-- Hangman Sessions
CREATE TABLE IF NOT EXISTS hangman_sessions (
    id SERIAL PRIMARY KEY,
    kid_id INTEGER NOT NULL,
    spelling_list_id INTEGER NOT NULL,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    total_games INTEGER NOT NULL,
    games_won INTEGER DEFAULT 0,
    total_points INTEGER DEFAULT 0,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_hangman_sessions_kid ON hangman_sessions(kid_id);
CREATE INDEX IF NOT EXISTS idx_hangman_sessions_list ON hangman_sessions(spelling_list_id);

-- Hangman Games
CREATE TABLE IF NOT EXISTS hangman_games (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL,
    kid_id INTEGER NOT NULL,
    word_id INTEGER NOT NULL,
    word TEXT NOT NULL,
    guessed_letters TEXT NOT NULL DEFAULT '[]',
    wrong_guesses INTEGER DEFAULT 0,
    max_wrong_guesses INTEGER DEFAULT 6,
    is_won BOOLEAN DEFAULT FALSE,
    is_lost BOOLEAN DEFAULT FALSE,
    points_earned INTEGER DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES hangman_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_hangman_games_session ON hangman_games(session_id);
CREATE INDEX IF NOT EXISTS idx_hangman_games_kid ON hangman_games(kid_id);

-- Hangman State (for persisting game progress)
CREATE TABLE IF NOT EXISTS hangman_state (
    kid_id INTEGER PRIMARY KEY,
    session_id INTEGER NOT NULL,
    current_word_idx INTEGER DEFAULT 0,
    words_json TEXT NOT NULL,
    points_so_far INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES hangman_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_hangman_state_session ON hangman_state(session_id);
