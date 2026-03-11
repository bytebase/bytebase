-- Add resource_id column to API-exposed tables that currently use integer IDs.
-- Use gen_random_uuid() as default for existing rows and new inserts.

ALTER TABLE plan ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE task ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE task_run ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE issue ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE issue_comment ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE revision ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE changelog ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE worksheet ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;
ALTER TABLE project_webhook ADD COLUMN resource_id text NOT NULL DEFAULT gen_random_uuid()::text;

CREATE UNIQUE INDEX idx_plan_unique_resource_id ON plan(resource_id);
CREATE UNIQUE INDEX idx_task_unique_resource_id ON task(resource_id);
CREATE UNIQUE INDEX idx_task_run_unique_resource_id ON task_run(resource_id);
CREATE UNIQUE INDEX idx_issue_unique_resource_id ON issue(resource_id);
CREATE UNIQUE INDEX idx_issue_comment_unique_resource_id ON issue_comment(resource_id);
CREATE UNIQUE INDEX idx_revision_unique_resource_id ON revision(resource_id);
CREATE UNIQUE INDEX idx_changelog_unique_resource_id ON changelog(resource_id);
CREATE UNIQUE INDEX idx_worksheet_unique_resource_id ON worksheet(resource_id);
CREATE UNIQUE INDEX idx_project_webhook_unique_resource_id ON project_webhook(resource_id);
