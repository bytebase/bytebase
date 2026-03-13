-- Phase A: Drop all FKs referencing plan chain tables
ALTER TABLE plan_check_run DROP CONSTRAINT IF EXISTS plan_check_run_plan_id_fkey;
ALTER TABLE plan_webhook_delivery DROP CONSTRAINT IF EXISTS plan_webhook_delivery_plan_id_fkey;
ALTER TABLE task DROP CONSTRAINT IF EXISTS task_plan_id_fkey;
ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_plan_id_fkey;
ALTER TABLE task_run DROP CONSTRAINT IF EXISTS task_run_task_id_fkey;
ALTER TABLE task_run_log DROP CONSTRAINT IF EXISTS task_run_log_task_run_id_fkey;

-- Phase B: Add project columns and change PKs (top-down)

-- plan (already has project, just change PK)
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'plan_pkey'
          AND conrelid = 'plan'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE plan DROP CONSTRAINT IF EXISTS plan_pkey;
        ALTER TABLE plan ADD PRIMARY KEY (project, id);
    END IF;
END $$;

-- task
ALTER TABLE task ADD COLUMN IF NOT EXISTS project TEXT;
UPDATE task SET project = plan.project FROM plan WHERE task.project IS NULL AND task.plan_id = plan.id;
ALTER TABLE task ALTER COLUMN project SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'task_pkey'
          AND conrelid = 'task'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE task DROP CONSTRAINT IF EXISTS task_pkey;
        ALTER TABLE task ADD PRIMARY KEY (project, id);
    END IF;
END $$;

-- task_run
ALTER TABLE task_run ADD COLUMN IF NOT EXISTS project TEXT;
UPDATE task_run SET project = task.project FROM task WHERE task_run.project IS NULL AND task_run.task_id = task.id;
ALTER TABLE task_run ALTER COLUMN project SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'task_run_pkey'
          AND conrelid = 'task_run'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE task_run DROP CONSTRAINT IF EXISTS task_run_pkey;
        ALTER TABLE task_run ADD PRIMARY KEY (project, id);
    END IF;
END $$;

-- task_run_log
ALTER TABLE task_run_log ADD COLUMN IF NOT EXISTS project TEXT;
UPDATE task_run_log SET project = task_run.project FROM task_run WHERE task_run_log.project IS NULL AND task_run_log.task_run_id = task_run.id;
ALTER TABLE task_run_log ALTER COLUMN project SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'task_run_log_pkey'
          AND conrelid = 'task_run_log'::regclass
          AND array_length(conkey, 1) > 1
    ) THEN
        ALTER TABLE task_run_log DROP CONSTRAINT IF EXISTS task_run_log_pkey;
        ALTER TABLE task_run_log ADD PRIMARY KEY (project, task_run_id, created_at);
    END IF;
END $$;

-- plan_check_run
ALTER TABLE plan_check_run ADD COLUMN IF NOT EXISTS project TEXT;
UPDATE plan_check_run SET project = plan.project FROM plan WHERE plan_check_run.project IS NULL AND plan_check_run.plan_id = plan.id;
ALTER TABLE plan_check_run ALTER COLUMN project SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'plan_check_run_pkey'
          AND conrelid = 'plan_check_run'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE plan_check_run DROP CONSTRAINT IF EXISTS plan_check_run_pkey;
        ALTER TABLE plan_check_run ADD PRIMARY KEY (project, id);
    END IF;
END $$;

-- plan_webhook_delivery
ALTER TABLE plan_webhook_delivery ADD COLUMN IF NOT EXISTS project TEXT;
UPDATE plan_webhook_delivery SET project = plan.project FROM plan WHERE plan_webhook_delivery.project IS NULL AND plan_webhook_delivery.plan_id = plan.id;
DELETE FROM plan_webhook_delivery WHERE project IS NULL;
ALTER TABLE plan_webhook_delivery ALTER COLUMN project SET NOT NULL;
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'plan_webhook_delivery_pkey'
          AND conrelid = 'plan_webhook_delivery'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE plan_webhook_delivery DROP CONSTRAINT IF EXISTS plan_webhook_delivery_pkey;
        ALTER TABLE plan_webhook_delivery ADD PRIMARY KEY (project, plan_id);
    END IF;
END $$;

-- Phase C: Re-add composite FKs and update indexes

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_plan_id_fkey' AND conrelid = 'task'::regclass) THEN
        ALTER TABLE task ADD CONSTRAINT task_plan_id_fkey
            FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_run_task_id_fkey' AND conrelid = 'task_run'::regclass) THEN
        ALTER TABLE task_run ADD CONSTRAINT task_run_task_id_fkey
            FOREIGN KEY (project, task_id) REFERENCES task(project, id);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'task_run_log_task_run_id_fkey' AND conrelid = 'task_run_log'::regclass) THEN
        ALTER TABLE task_run_log ADD CONSTRAINT task_run_log_task_run_id_fkey
            FOREIGN KEY (project, task_run_id) REFERENCES task_run(project, id);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plan_check_run_plan_id_fkey' AND conrelid = 'plan_check_run'::regclass) THEN
        ALTER TABLE plan_check_run ADD CONSTRAINT plan_check_run_plan_id_fkey
            FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plan_webhook_delivery_plan_id_fkey' AND conrelid = 'plan_webhook_delivery'::regclass) THEN
        ALTER TABLE plan_webhook_delivery ADD CONSTRAINT plan_webhook_delivery_plan_id_fkey
            FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'issue_plan_id_fkey' AND conrelid = 'issue'::regclass) THEN
        ALTER TABLE issue ADD CONSTRAINT issue_plan_id_fkey
            FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);
    END IF;
END $$;

-- Update indexes to include project
DROP INDEX IF EXISTS idx_task_plan_id_environment;
CREATE INDEX IF NOT EXISTS idx_task_plan_id_environment ON task(project, plan_id, environment);

DROP INDEX IF EXISTS uk_task_run_task_id_attempt;
CREATE UNIQUE INDEX IF NOT EXISTS uk_task_run_task_id_attempt ON task_run(project, task_id, attempt);

DROP INDEX IF EXISTS idx_plan_check_run_unique_plan_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_plan_check_run_unique_plan_id ON plan_check_run(project, plan_id);

DROP INDEX IF EXISTS idx_issue_unique_plan_id;
CREATE UNIQUE INDEX IF NOT EXISTS idx_issue_unique_plan_id ON issue(project, plan_id);
