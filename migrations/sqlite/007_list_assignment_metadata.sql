-- Add teacher-managed and due date metadata to list assignments

ALTER TABLE list_assignments ADD COLUMN managed_by_teacher BOOLEAN DEFAULT 0 NOT NULL;
ALTER TABLE list_assignments ADD COLUMN due_date DATETIME;
