-- SpellingClash Database Schema (SQLite)
-- Consolidated schema for fresh installations

-- Users (Parents)
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Sessions table for web authentication
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Families
CREATE TABLE IF NOT EXISTS families (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Family Members
CREATE TABLE IF NOT EXISTS family_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    role TEXT DEFAULT 'parent' CHECK(role IN ('parent', 'admin')),
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(family_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_family_members_family ON family_members(family_id);
CREATE INDEX IF NOT EXISTS idx_family_members_user ON family_members(user_id);

-- Kids
CREATE TABLE IF NOT EXISTS kids (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    username TEXT,
    password TEXT,
    avatar_color TEXT DEFAULT '#4A90E2',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_kids_family ON kids(family_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_kids_username ON kids(username);

-- Kid Sessions
CREATE TABLE IF NOT EXISTS kid_sessions (
    id TEXT PRIMARY KEY,
    kid_id INTEGER NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_kid_sessions_kid ON kid_sessions(kid_id);
CREATE INDEX IF NOT EXISTS idx_kid_sessions_expires ON kid_sessions(expires_at);

-- Spelling Lists
CREATE TABLE IF NOT EXISTS spelling_lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_id INTEGER,
    name TEXT NOT NULL,
    description TEXT,
    created_by INTEGER NOT NULL,
    is_public BOOLEAN DEFAULT 0 NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_spelling_lists_family ON spelling_lists(family_id);
CREATE INDEX IF NOT EXISTS idx_spelling_lists_public ON spelling_lists(is_public);

-- Words
CREATE TABLE IF NOT EXISTS words (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spelling_list_id INTEGER NOT NULL,
    word_text TEXT NOT NULL,
    difficulty_level INTEGER DEFAULT 1 CHECK(difficulty_level BETWEEN 1 AND 5),
    audio_filename TEXT,
    definition TEXT,
    definition_audio_filename TEXT,
    position INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_words_list ON words(spelling_list_id);
CREATE INDEX IF NOT EXISTS idx_words_position ON words(spelling_list_id, position);

-- List Assignments
CREATE TABLE IF NOT EXISTS list_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spelling_list_id INTEGER NOT NULL,
    kid_id INTEGER NOT NULL,
    assigned_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    assigned_by INTEGER NOT NULL,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(spelling_list_id, kid_id)
);

CREATE INDEX IF NOT EXISTS idx_list_assignments_kid ON list_assignments(kid_id);
CREATE INDEX IF NOT EXISTS idx_list_assignments_list ON list_assignments(spelling_list_id);

-- Practice Sessions
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

CREATE INDEX IF NOT EXISTS idx_practice_sessions_kid ON practice_sessions(kid_id);
CREATE INDEX IF NOT EXISTS idx_practice_sessions_list ON practice_sessions(spelling_list_id);

-- Word Attempts
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

CREATE INDEX IF NOT EXISTS idx_word_attempts_session ON word_attempts(practice_session_id);
CREATE INDEX IF NOT EXISTS idx_word_attempts_word ON word_attempts(word_id);

-- Practice State
CREATE TABLE IF NOT EXISTS practice_state (
    kid_id INTEGER PRIMARY KEY,
    session_id INTEGER NOT NULL,
    current_index INTEGER NOT NULL DEFAULT 0,
    correct_count INTEGER NOT NULL DEFAULT 0,
    total_points INTEGER NOT NULL DEFAULT 0,
    start_time DATETIME NOT NULL,
    word_order TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES practice_sessions(id) ON DELETE CASCADE
);

-- Practice Word Timing
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
