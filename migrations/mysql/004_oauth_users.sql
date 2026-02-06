-- Add OAuth provider fields to users
ALTER TABLE users ADD COLUMN oauth_provider VARCHAR(50) NULL;
ALTER TABLE users ADD COLUMN oauth_subject VARCHAR(255) NULL;

CREATE UNIQUE INDEX idx_users_oauth ON users(oauth_provider, oauth_subject);
