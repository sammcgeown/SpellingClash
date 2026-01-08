-- Add Spelling Lists and Words tables
-- Phase 3: Spelling Lists

-- Spelling Lists
CREATE TABLE IF NOT EXISTS spelling_lists (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_spelling_lists_family ON spelling_lists(family_id);

-- Words in spelling lists
CREATE TABLE IF NOT EXISTS words (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spelling_list_id INTEGER NOT NULL,
    word_text TEXT NOT NULL,
    difficulty_level INTEGER DEFAULT 1 CHECK(difficulty_level BETWEEN 1 AND 5),
    audio_filename TEXT,
    position INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (spelling_list_id) REFERENCES spelling_lists(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_words_list ON words(spelling_list_id);
CREATE INDEX IF NOT EXISTS idx_words_position ON words(spelling_list_id, position);

-- List assignments to kids
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
