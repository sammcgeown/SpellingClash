-- Add family_code column to families table if it doesn't exist

-- Check and add column if needed
SET @dbname = DATABASE();
SET @tablename = 'families';
SET @columnname = 'family_code';
SET @preparedStatement = (SELECT IF(
  (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
   WHERE TABLE_SCHEMA = @dbname AND TABLE_NAME = @tablename AND COLUMN_NAME = @columnname) > 0,
  'SELECT 1',
  'ALTER TABLE families ADD COLUMN family_code VARCHAR(255)'
));
PREPARE alterIfNotExists FROM @preparedStatement;
EXECUTE alterIfNotExists;
DEALLOCATE PREPARE alterIfNotExists;

-- Update existing families with generated codes (8 character hex)
UPDATE families 
SET family_code = LOWER(HEX(RANDOM_BYTES(4)))
WHERE family_code IS NULL OR family_code = '';

-- Create index on family_code
CREATE INDEX idx_families_code ON families(family_code);
