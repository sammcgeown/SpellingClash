-- Settings table for app-wide configuration
CREATE TABLE IF NOT EXISTS settings (
    `key` VARCHAR(255) PRIMARY KEY,
    `value` TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Insert default setting for invite-only mode (disabled by default)
INSERT IGNORE INTO settings (`key`, `value`) VALUES ('invite_only_mode', 'false');

-- Invitations table
CREATE TABLE IF NOT EXISTS invitations (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    code VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    invited_by INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    used_at TIMESTAMP NULL,
    used_by INTEGER NULL,
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (used_by) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_invitations_code ON invitations(code);
CREATE INDEX idx_invitations_email ON invitations(email);
CREATE INDEX idx_invitations_used_at ON invitations(used_at);
