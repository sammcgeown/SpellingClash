-- Add teacher-managed and due date metadata to list assignments

ALTER TABLE list_assignments ADD COLUMN IF NOT EXISTS managed_by_teacher BOOLEAN DEFAULT FALSE NOT NULL;
ALTER TABLE list_assignments ADD COLUMN IF NOT EXISTS due_date TIMESTAMP;
