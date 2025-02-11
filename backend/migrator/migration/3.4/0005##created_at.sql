ALTER TABLE task_run_log RENAME COLUMN created_ts TO created_at;
DROP INDEX IF EXISTS idx_revision_unique_database_id_version_deleted_ts_null;
ALTER TABLE revision RENAME COLUMN created_ts TO created_at;
ALTER TABLE revision RENAME COLUMN deleted_ts TO deleted_at;
CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_unique_database_id_version_deleted_at_null ON revision (database_id, version) WHERE deleted_at IS NULL;
ALTER TABLE sync_history RENAME COLUMN created_ts TO created_at;
ALTER TABLE changelog RENAME COLUMN created_ts TO created_at;
ALTER TABLE release RENAME COLUMN created_ts TO created_at;
ALTER TABLE policy RENAME COLUMN updated_ts TO updated_at;
