-- Add username and password to kids table
-- Kids should have randomly generated credentials for login

ALTER TABLE kids ADD COLUMN username TEXT;
ALTER TABLE kids ADD COLUMN password TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_kids_username ON kids(username);
