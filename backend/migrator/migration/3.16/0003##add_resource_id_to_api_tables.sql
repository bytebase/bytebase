-- Add resource_id column to API-exposed tables that currently use integer IDs.
-- Use gen_random_uuid() as default for existing rows and new inserts.

DO $$
DECLARE
    tables text[] := ARRAY['plan', 'task', 'task_run', 'issue', 'issue_comment', 'revision', 'changelog', 'worksheet', 'project_webhook'];
    t text;
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = t AND column_name = 'resource_id'
        ) THEN
            EXECUTE format('ALTER TABLE %I ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text', t);
        END IF;
    END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_unique_resource_id ON plan(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_task_unique_resource_id ON task(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_task_run_unique_resource_id ON task_run(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_issue_unique_resource_id ON issue(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_issue_comment_unique_resource_id ON issue_comment(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_revision_unique_resource_id ON revision(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_changelog_unique_resource_id ON changelog(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_worksheet_unique_resource_id ON worksheet(resource_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_project_webhook_unique_resource_id ON project_webhook(resource_id);
