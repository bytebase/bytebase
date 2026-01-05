-- Remove code and sheet_sha256 columns from task_run table
-- These fields are no longer used:
-- - code: was write-only, never read
-- - sheet_sha256: redundant denormalization of task.payload.sheet_sha256

ALTER TABLE task_run DROP COLUMN code;
ALTER TABLE task_run DROP COLUMN sheet_sha256;
