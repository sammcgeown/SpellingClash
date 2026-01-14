-- Hangman game tables (MySQL)

-- Hangman Sessions
CREATE TABLE IF NOT EXISTS hangman_sessions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    kid_id INT NOT NULL,
    spelling_list_id INT NOT NULL,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    total_games INT NOT NULL,
    games_won INT DEFAULT 0,
    total_points INT DEFAULT 0,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE,
    INDEX idx_hangman_sessions_kid (kid_id),
    INDEX idx_hangman_sessions_list (spelling_list_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Hangman Games
CREATE TABLE IF NOT EXISTS hangman_games (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id INT NOT NULL,
    kid_id INT NOT NULL,
    word_id INT NOT NULL,
    word TEXT NOT NULL,
    guessed_letters TEXT NOT NULL,
    wrong_guesses INT DEFAULT 0,
    max_wrong_guesses INT DEFAULT 6,
    is_won BOOLEAN DEFAULT FALSE,
    is_lost BOOLEAN DEFAULT FALSE,
    points_earned INT DEFAULT 0,
    started_at DATETIME NOT NULL,
    completed_at DATETIME,
    FOREIGN KEY (session_id) REFERENCES hangman_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE,
    INDEX idx_hangman_games_session (session_id),
    INDEX idx_hangman_games_kid (kid_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Hangman State (for persisting game progress)
CREATE TABLE IF NOT EXISTS hangman_state (
    kid_id INT PRIMARY KEY,
    session_id INT NOT NULL,
    current_word_idx INT DEFAULT 0,
    words_json TEXT NOT NULL,
    points_so_far INT DEFAULT 0,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES hangman_sessions(id) ON DELETE CASCADE,
    INDEX idx_hangman_state_session (session_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
