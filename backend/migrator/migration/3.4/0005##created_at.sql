ALTER TABLE task_run_log RENAME COLUMN created_ts TO created_at;
ALTER TABLE revision RENAME COLUMN created_ts TO created_at;
ALTER TABLE revision RENAME COLUMN deleted_ts TO deleted_at;
ALTER TABLE sync_history RENAME COLUMN created_ts TO created_at;
ALTER TABLE changelog RENAME COLUMN created_ts TO created_at;
ALTER TABLE release RENAME COLUMN created_ts TO created_at;
ALTER TABLE policy RENAME COLUMN updated_ts TO updated_at;
