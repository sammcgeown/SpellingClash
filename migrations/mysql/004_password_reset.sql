-- Add password reset tokens table
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token VARCHAR(255) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for faster lookups by user_id
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);

-- Index for cleanup of expired tokens
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at);
