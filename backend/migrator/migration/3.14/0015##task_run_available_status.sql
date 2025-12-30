-- Add AVAILABLE status to task_run CHECK constraint
ALTER TABLE task_run DROP CONSTRAINT task_run_status_check;
ALTER TABLE task_run ADD CONSTRAINT task_run_status_check
    CHECK (status IN ('PENDING', 'AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));

-- Update partial index to include AVAILABLE
DROP INDEX idx_task_run_active_status_id;
CREATE INDEX idx_task_run_active_status_id ON task_run(status, id)
    WHERE status IN ('PENDING', 'AVAILABLE', 'RUNNING');
