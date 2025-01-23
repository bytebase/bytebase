ALTER TABLE task_run DISABLE TRIGGER update_task_run_updated_ts;
UPDATE task_run SET result = result - 'migrationId' WHERE result ? 'migrationId';
ALTER TABLE task_run ENABLE TRIGGER update_task_run_updated_ts;
