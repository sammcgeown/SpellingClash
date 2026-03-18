-- Add teacher account support and teacher-child relationships

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_teacher BOOLEAN DEFAULT FALSE NOT NULL;

CREATE TABLE IF NOT EXISTS teacher_kid_relationships (
    id SERIAL PRIMARY KEY,
    teacher_user_id BIGINT NOT NULL,
    kid_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (teacher_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (kid_id) REFERENCES kids(id) ON DELETE CASCADE,
    UNIQUE(teacher_user_id, kid_id)
);

CREATE INDEX IF NOT EXISTS idx_teacher_kid_relationships_teacher ON teacher_kid_relationships(teacher_user_id);
CREATE INDEX IF NOT EXISTS idx_teacher_kid_relationships_kid ON teacher_kid_relationships(kid_id);
