-- Add teacher-managed and due date metadata to list assignments

ALTER TABLE list_assignments
    ADD COLUMN managed_by_teacher BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN due_date DATETIME NULL;
