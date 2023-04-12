ALTER TABLE pipeline DROP COLUMN status;

DROP INDEX IF EXISTS idx_pipeline_status;