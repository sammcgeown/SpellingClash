-- Add family_code column to families table if it doesn't exist

-- Add the column (PostgreSQL supports IF NOT EXISTS for columns via a DO block)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'families' AND column_name = 'family_code') THEN
        ALTER TABLE families ADD COLUMN family_code TEXT;
    END IF;
END $$;

-- Update existing families with generated codes (8 character hex)
UPDATE families 
SET family_code = encode(gen_random_bytes(4), 'hex')
WHERE family_code IS NULL OR family_code = '';

-- Make family_code NOT NULL and UNIQUE after populating
ALTER TABLE families ALTER COLUMN family_code SET NOT NULL;

-- Create unique index on family_code
CREATE INDEX IF NOT EXISTS idx_families_code ON families(family_code);
