DROP INDEX IF EXISTS idx_issue_pipeline_id;
ALTER TABLE issue DROP COLUMN IF EXISTS pipeline_id;
