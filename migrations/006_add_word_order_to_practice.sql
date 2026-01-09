-- Add word order column to store randomized word IDs for practice sessions
ALTER TABLE practice_state ADD COLUMN word_order TEXT;

-- word_order will store comma-separated word IDs in the randomized order
