ALTER TABLE task_run DISABLE TRIGGER update_task_run_updated_ts;

ALTER TABLE task_run ADD COLUMN started_ts BIGINT NOT NULL DEFAULT 0;

UPDATE task_run
SET started_ts = created_ts;

ALTER TABLE task_run ENABLE TRIGGER update_task_run_updated_ts;
