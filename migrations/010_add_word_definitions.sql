-- Add optional definition field to words
-- Definitions help provide context for spelling practice

ALTER TABLE words ADD COLUMN definition TEXT;
ALTER TABLE words ADD COLUMN definition_audio_filename TEXT;
