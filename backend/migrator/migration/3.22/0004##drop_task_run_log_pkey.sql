-- task_run_log is an append-only log with no natural key: entries for one task
-- run can legitimately share a created_at microsecond, and legacy data already
-- does (BYT-10035). Databases that upgraded through the original 3.16.2 carry
-- PRIMARY KEY (task_run_id, created_at) while fresh installs carry
-- PRIMARY KEY (project, task_run_id, created_at); drop whichever is present and
-- index the read path instead.
ALTER TABLE task_run_log DROP CONSTRAINT IF EXISTS task_run_log_pkey;
DROP INDEX IF EXISTS idx_task_run_log_task_run_id;
DROP INDEX IF EXISTS idx_task_run_log_task_run_id_created_at;
CREATE INDEX IF NOT EXISTS idx_task_run_log_project_task_run_id_created_at ON task_run_log(project, task_run_id, created_at);
