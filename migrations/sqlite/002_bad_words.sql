-- Bad words filter table
CREATE TABLE IF NOT EXISTS bad_words (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    word TEXT UNIQUE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bad_words_word ON bad_words(word);
