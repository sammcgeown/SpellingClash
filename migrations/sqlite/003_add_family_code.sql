-- Add family_code column to families table if it doesn't exist
-- SQLite doesn't have IF NOT EXISTS for ALTER TABLE, so we need to handle this carefully

-- Check if family_code column exists, if not add it
-- This uses a pragma check to see if the column exists
-- For SQLite, we'll just add the column and handle the error if it already exists

-- First, add the column (will fail silently if already exists due to our migration tracking)
ALTER TABLE families ADD COLUMN family_code TEXT;

-- Update existing families with generated codes (8 character hex)
UPDATE families 
SET family_code = lower(hex(randomblob(4)))
WHERE family_code IS NULL OR family_code = '';

-- Create index on family_code
CREATE INDEX IF NOT EXISTS idx_families_code ON families(family_code);
