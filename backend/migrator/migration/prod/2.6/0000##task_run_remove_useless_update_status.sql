ALTER TABLE task_run DROP IF EXISTS type, DROP IF EXISTS comment, DROP IF EXISTS payload;

ALTER TABLE task_run DROP CONSTRAINT task_run_status_check;

ALTER TABLE task_run ADD CONSTRAINT task_run_status_check CHECK (status IN ('PENDING', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));