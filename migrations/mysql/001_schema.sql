-- SpellingClash Database Schema (MySQL)
-- Consolidated schema for fresh installations

-- Users (Parents)
CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_admin BOOLEAN DEFAULT 0 NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_admin ON users(is_admin);

-- Sessions table for web authentication
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Families
CREATE TABLE IF NOT EXISTS families (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Family Members
CREATE TABLE IF NOT EXISTS family_members (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    family_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role VARCHAR(50) DEFAULT 'parent',
    joined_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_family_user (family_id, user_id),
    CHECK (role IN ('parent', 'admin'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_family_members_family ON family_members(family_id);
CREATE INDEX idx_family_members_user ON family_members(user_id);

-- Kids
CREATE TABLE IF NOT EXISTS kids (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    family_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255),
    password VARCHAR(255),
    avatar_color VARCHAR(20) DEFAULT '#4A90E2',
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    UNIQUE KEY idx_kids_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_kids_family ON kids(family_id);

-- Kid Sessions
CREATE TABLE IF NOT EXISTS kid_sessions (
    id VARCHAR(255) PRIMARY KEY,
    kid_id BIGINT NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_kid_sessions_kid ON kid_sessions(kid_id);
CREATE INDEX idx_kid_sessions_expires ON kid_sessions(expires_at);

-- Spelling Lists
CREATE TABLE IF NOT EXISTS spelling_lists (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    family_id BIGINT,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by BIGINT,
    is_public BOOLEAN DEFAULT FALSE NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_spelling_lists_family ON spelling_lists(family_id);
CREATE INDEX idx_spelling_lists_public ON spelling_lists(is_public);

-- Words
CREATE TABLE IF NOT EXISTS words (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    spelling_list_id BIGINT NOT NULL,
    word_text VARCHAR(255) NOT NULL,
    difficulty_level INTEGER DEFAULT 1,
    audio_filename VARCHAR(255),
    definition TEXT,
    definition_audio_filename VARCHAR(255),
    position INTEGER NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE,
    CHECK (difficulty_level BETWEEN 1 AND 5)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_words_list ON words(spelling_list_id);
CREATE INDEX idx_words_position ON words(spelling_list_id, position);

-- List Assignments
CREATE TABLE IF NOT EXISTS list_assignments (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    spelling_list_id BIGINT NOT NULL,
    kid_id BIGINT NOT NULL,
    assigned_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
    assigned_by BIGINT NOT NULL,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_list_kid (spelling_list_id, kid_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_list_assignments_kid ON list_assignments(kid_id);
CREATE INDEX idx_list_assignments_list ON list_assignments(spelling_list_id);

-- Practice Sessions
CREATE TABLE IF NOT EXISTS practice_sessions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    kid_id BIGINT NOT NULL,
    spelling_list_id BIGINT NOT NULL,
    started_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    completed_at DATETIME(6),
    total_words INTEGER NOT NULL DEFAULT 0,
    correct_words INTEGER NOT NULL DEFAULT 0,
    points_earned INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_practice_sessions_kid ON practice_sessions(kid_id);
CREATE INDEX idx_practice_sessions_list ON practice_sessions(spelling_list_id);

-- Word Attempts
CREATE TABLE IF NOT EXISTS word_attempts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    practice_session_id BIGINT NOT NULL,
    word_id BIGINT NOT NULL,
    attempt_text TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    time_taken_ms INTEGER NOT NULL,
    points_earned INTEGER NOT NULL DEFAULT 0,
    attempted_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    FOREIGN KEY (practice_session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE,
    FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_word_attempts_session ON word_attempts(practice_session_id);
CREATE INDEX idx_word_attempts_word ON word_attempts(word_id);

-- Practice State
CREATE TABLE IF NOT EXISTS practice_state (
    kid_id BIGINT PRIMARY KEY,
    session_id BIGINT NOT NULL,
    current_index INTEGER NOT NULL DEFAULT 0,
    correct_count INTEGER NOT NULL DEFAULT 0,
    total_points INTEGER NOT NULL DEFAULT 0,
    start_time DATETIME(6) NOT NULL,
    word_order TEXT,
    updated_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Practice Word Timing
CREATE TABLE IF NOT EXISTS practice_word_timing (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    kid_id BIGINT NOT NULL,
    session_id BIGINT NOT NULL,
    word_index INTEGER NOT NULL,
    started_at DATETIME(6) NOT NULL,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE,
    UNIQUE KEY unique_kid_session_word (kid_id, session_id, word_index)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_practice_word_timing_kid_session ON practice_word_timing(kid_id, session_id);

