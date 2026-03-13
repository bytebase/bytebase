-- Phase A: Drop all FKs referencing plan chain tables

-- FKs referencing plan(id)
ALTER TABLE plan_check_run DROP CONSTRAINT plan_check_run_plan_id_fkey;
ALTER TABLE plan_webhook_delivery DROP CONSTRAINT plan_webhook_delivery_plan_id_fkey;
ALTER TABLE task DROP CONSTRAINT task_plan_id_fkey;
ALTER TABLE issue DROP CONSTRAINT issue_plan_id_fkey;
-- FKs referencing task(id)
ALTER TABLE task_run DROP CONSTRAINT task_run_task_id_fkey;
-- FKs referencing task_run(id)
ALTER TABLE task_run_log DROP CONSTRAINT task_run_log_task_run_id_fkey;

-- Phase B: Add project columns and change PKs (top-down)

-- plan (already has project, just change PK)
ALTER TABLE plan DROP CONSTRAINT plan_pkey;
ALTER TABLE plan ADD PRIMARY KEY (project, id);

-- task
ALTER TABLE task ADD COLUMN project TEXT;
UPDATE task SET project = plan.project FROM plan WHERE task.plan_id = plan.id;
ALTER TABLE task ALTER COLUMN project SET NOT NULL;
ALTER TABLE task DROP CONSTRAINT task_pkey;
ALTER TABLE task ADD PRIMARY KEY (project, id);

-- task_run
ALTER TABLE task_run ADD COLUMN project TEXT;
UPDATE task_run SET project = task.project FROM task WHERE task_run.task_id = task.id;
ALTER TABLE task_run ALTER COLUMN project SET NOT NULL;
ALTER TABLE task_run DROP CONSTRAINT task_run_pkey;
ALTER TABLE task_run ADD PRIMARY KEY (project, id);

-- task_run_log
ALTER TABLE task_run_log ADD COLUMN project TEXT;
UPDATE task_run_log SET project = task_run.project FROM task_run WHERE task_run_log.task_run_id = task_run.id;
ALTER TABLE task_run_log ALTER COLUMN project SET NOT NULL;
ALTER TABLE task_run_log DROP CONSTRAINT task_run_log_pkey;
ALTER TABLE task_run_log ADD PRIMARY KEY (project, task_run_id, created_at);

-- plan_check_run
ALTER TABLE plan_check_run ADD COLUMN project TEXT;
UPDATE plan_check_run SET project = plan.project FROM plan WHERE plan_check_run.plan_id = plan.id;
ALTER TABLE plan_check_run ALTER COLUMN project SET NOT NULL;
ALTER TABLE plan_check_run DROP CONSTRAINT plan_check_run_pkey;
ALTER TABLE plan_check_run ADD PRIMARY KEY (project, id);

-- plan_webhook_delivery
ALTER TABLE plan_webhook_delivery ADD COLUMN project TEXT;
UPDATE plan_webhook_delivery SET project = plan.project FROM plan WHERE plan_webhook_delivery.plan_id = plan.id;
-- Handle orphan rows (plan_id with no matching plan)
DELETE FROM plan_webhook_delivery WHERE project IS NULL;
ALTER TABLE plan_webhook_delivery ALTER COLUMN project SET NOT NULL;
ALTER TABLE plan_webhook_delivery DROP CONSTRAINT plan_webhook_delivery_pkey;
ALTER TABLE plan_webhook_delivery ADD PRIMARY KEY (project, plan_id);

-- Phase C: Re-add composite FKs and update indexes

-- task -> plan
ALTER TABLE task ADD CONSTRAINT task_plan_id_fkey
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);

-- task_run -> task
ALTER TABLE task_run ADD CONSTRAINT task_run_task_id_fkey
    FOREIGN KEY (project, task_id) REFERENCES task(project, id);

-- task_run_log -> task_run
ALTER TABLE task_run_log ADD CONSTRAINT task_run_log_task_run_id_fkey
    FOREIGN KEY (project, task_run_id) REFERENCES task_run(project, id);

-- plan_check_run -> plan
ALTER TABLE plan_check_run ADD CONSTRAINT plan_check_run_plan_id_fkey
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);

-- plan_webhook_delivery -> plan
ALTER TABLE plan_webhook_delivery ADD CONSTRAINT plan_webhook_delivery_plan_id_fkey
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);

-- issue -> plan (cross-chain, issue already has project column)
ALTER TABLE issue ADD CONSTRAINT issue_plan_id_fkey
    FOREIGN KEY (project, plan_id) REFERENCES plan(project, id);

-- Update indexes to include project
DROP INDEX idx_task_plan_id_environment;
CREATE INDEX idx_task_plan_id_environment ON task(project, plan_id, environment);

DROP INDEX uk_task_run_task_id_attempt;
CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run(project, task_id, attempt);

DROP INDEX idx_plan_check_run_unique_plan_id;
CREATE UNIQUE INDEX idx_plan_check_run_unique_plan_id ON plan_check_run(project, plan_id);

-- issue unique plan_id index
DROP INDEX idx_issue_unique_plan_id;
CREATE UNIQUE INDEX idx_issue_unique_plan_id ON issue(project, plan_id);
