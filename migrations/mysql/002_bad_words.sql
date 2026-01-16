-- Bad words filter table
CREATE TABLE IF NOT EXISTS bad_words (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    word VARCHAR(255) UNIQUE NOT NULL,
    created_at DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_bad_words_word ON bad_words(word);
