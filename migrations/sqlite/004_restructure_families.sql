-- Restructure families table to use family_code as primary key
-- SQLite doesn't support dropping columns or changing primary keys directly,
-- so we need to create new tables and migrate data

-- CRITICAL: Disable foreign key checks during migration
PRAGMA foreign_keys = OFF;

-- Step 0: Clean up any partial migration attempts
DROP TABLE IF EXISTS families_new;
DROP TABLE IF EXISTS family_members_new;
DROP TABLE IF EXISTS kids_new;
DROP TABLE IF EXISTS spelling_lists_new;

-- Step 1: Create new families table with family_code as primary key
CREATE TABLE families_new (
    family_code TEXT PRIMARY KEY,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Step 2: Copy data from old families table
INSERT INTO families_new (family_code, created_at, updated_at)
SELECT family_code, created_at, updated_at FROM families;

-- Step 3: Create new family_members table referencing family_code
CREATE TABLE family_members_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_code TEXT NOT NULL,
    user_id INTEGER NOT NULL,
    role TEXT DEFAULT 'parent' CHECK(role IN ('parent', 'admin')),
    joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_code) REFERENCES families(family_code) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(family_code, user_id)
);

-- Step 4: Migrate family_members data
INSERT INTO family_members_new (family_code, user_id, role, joined_at)
SELECT f.family_code, fm.user_id, fm.role, fm.joined_at
FROM family_members fm
JOIN families f ON fm.family_id = f.id;

-- Step 5: Create new kids table referencing family_code
CREATE TABLE kids_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_code TEXT NOT NULL,
    name TEXT NOT NULL,
    username TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    avatar_color TEXT DEFAULT '#4A90E2',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (family_code) REFERENCES families(family_code) ON DELETE CASCADE
);

-- Step 6: Migrate kids data
INSERT INTO kids_new (id, family_code, name, username, password, avatar_color, created_at, updated_at)
SELECT k.id, f.family_code, k.name, k.username, k.password, k.avatar_color, k.created_at, k.updated_at
FROM kids k
JOIN families f ON k.family_id = f.id;

-- Step 7: Create new spelling_lists table with family_code
CREATE TABLE spelling_lists_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    family_code TEXT,
    name TEXT NOT NULL,
    description TEXT,
    created_by INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_public BOOLEAN DEFAULT 0,
    FOREIGN KEY (family_code) REFERENCES families(family_code) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Step 8: Migrate spelling_lists data
INSERT INTO spelling_lists_new (id, family_code, name, description, created_by, created_at, updated_at, is_public)
SELECT sl.id, f.family_code, sl.name, sl.description, sl.created_by, sl.created_at, sl.updated_at, sl.is_public
FROM spelling_lists sl
LEFT JOIN families f ON sl.family_id = f.id;

-- Step 9: Drop old tables
DROP TABLE family_members;
DROP TABLE kids;
DROP TABLE spelling_lists;
DROP TABLE families;

-- Step 10: Rename new tables
ALTER TABLE families_new RENAME TO families;
ALTER TABLE family_members_new RENAME TO family_members;
ALTER TABLE kids_new RENAME TO kids;
ALTER TABLE spelling_lists_new RENAME TO spelling_lists;

-- Step 11: Recreate indexes
CREATE INDEX idx_family_members_family ON family_members(family_code);
CREATE INDEX idx_family_members_user ON family_members(user_id);
CREATE INDEX idx_kids_family ON kids(family_code);
CREATE INDEX idx_kids_username ON kids(username);
CREATE INDEX idx_spelling_lists_family ON spelling_lists(family_code);

-- Re-enable foreign key checks
PRAGMA foreign_keys = ON;
