SET timezone = 'UTC';
ALTER TABLE task_run_log ALTER created_ts TYPE TIMESTAMPTZ;
RESET timezone;
