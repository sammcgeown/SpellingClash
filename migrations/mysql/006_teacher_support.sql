-- Add teacher account support and teacher-child relationships

ALTER TABLE users ADD COLUMN is_teacher BOOLEAN DEFAULT FALSE NOT NULL;

CREATE TABLE IF NOT EXISTS teacher_kid_relationships (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    teacher_user_id BIGINT NOT NULL,
    kid_id BIGINT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (teacher_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    UNIQUE KEY uk_teacher_kid (teacher_user_id, kid_id),
    INDEX idx_teacher_kid_relationships_teacher (teacher_user_id),
    INDEX idx_teacher_kid_relationships_kid (kid_id)
);
