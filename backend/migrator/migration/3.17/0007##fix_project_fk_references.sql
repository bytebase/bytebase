-- Add missing direct REFERENCES project(resource_id) to tables that only have composite FKs.
-- Rename query_history.project_id to project and add FK reference.

DO $$
DECLARE
    t text;
BEGIN
    -- Rename query_history.project_id to project before adding the FK.
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'query_history' AND column_name = 'project_id') THEN
        ALTER TABLE query_history RENAME COLUMN project_id TO project;
    END IF;

    -- Add direct project FK to tables that only had composite FKs.
    FOREACH t IN ARRAY ARRAY['plan_check_run', 'plan_webhook_delivery', 'task_run', 'task_run_log', 'issue_comment', 'query_history'] LOOP
        IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = t AND column_name = 'project')
           AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = t || '_project_fkey') THEN
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (project) REFERENCES project(resource_id)', t, t || '_project_fkey');
        END IF;
    END LOOP;
END $$;

-- Recreate the index with the new column name.
DROP INDEX IF EXISTS idx_query_history_creator_created_at_project_id;
CREATE INDEX IF NOT EXISTS idx_query_history_creator_created_at_project ON query_history(creator, created_at, project DESC);
