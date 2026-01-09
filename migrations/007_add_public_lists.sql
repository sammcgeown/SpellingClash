-- Add support for public/shared spelling lists
-- Public lists are available to all users and not tied to a specific family

-- Add is_public column to spelling_lists
ALTER TABLE spelling_lists ADD COLUMN is_public BOOLEAN DEFAULT 0 NOT NULL;

-- Make family_id nullable for public lists
-- SQLite doesn't support modifying columns directly, so we need to recreate the table
CREATE TABLE spelling_lists_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_id INTEGER,
    name TEXT NOT NULL,
    description TEXT,
    created_by INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_public BOOLEAN DEFAULT 0 NOT NULL,
    FOREIGN KEY (family_id) REFERENCES families(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
);

-- Copy data from old table
INSERT INTO spelling_lists_new (id, family_id, name, description, created_by, created_at, updated_at, is_public)
SELECT id, family_id, name, description, created_by, created_at, updated_at, 0
FROM spelling_lists;

-- Drop old table and rename new one
DROP TABLE spelling_lists;
ALTER TABLE spelling_lists_new RENAME TO spelling_lists;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_spelling_lists_family ON spelling_lists(family_id);
CREATE INDEX IF NOT EXISTS idx_spelling_lists_public ON spelling_lists(is_public);
