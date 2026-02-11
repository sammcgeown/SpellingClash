-- Add email status tracking to invitations
ALTER TABLE invitations ADD COLUMN email_sent BOOLEAN DEFAULT 1;
ALTER TABLE invitations ADD COLUMN email_error TEXT NULL;
ALTER TABLE invitations ADD COLUMN last_sent_at TIMESTAMP NULL;
