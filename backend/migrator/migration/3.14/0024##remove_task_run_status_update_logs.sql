-- Remove deprecated TASK_RUN_STATUS_UPDATE log entries.
-- These logs are no longer generated since the AVAILABLE status on task_run table
-- now captures the same semantic information.
DELETE FROM task_run_log WHERE payload->>'type' = 'TASK_RUN_STATUS_UPDATE';
