-- Migration: Convert FK columns from integer (referencing table.id) to text (referencing table.resource_id)
-- for Group B/C tables. Processes bottom-up through the dependency chain.

-----------------------
-- 1. worksheet_organizer.worksheet_id (int → text, references worksheet.resource_id, ON DELETE CASCADE)
-----------------------
ALTER TABLE worksheet_organizer DROP CONSTRAINT worksheet_organizer_worksheet_id_fkey;
DROP INDEX idx_worksheet_organizer_unique_sheet_id_principal;

ALTER TABLE worksheet_organizer ADD COLUMN new_worksheet_id text;
UPDATE worksheet_organizer wo SET new_worksheet_id = w.resource_id FROM worksheet w WHERE wo.worksheet_id = w.id;
ALTER TABLE worksheet_organizer DROP COLUMN worksheet_id;
ALTER TABLE worksheet_organizer RENAME COLUMN new_worksheet_id TO worksheet_id;
ALTER TABLE worksheet_organizer ALTER COLUMN worksheet_id SET NOT NULL;

ALTER TABLE worksheet_organizer ADD CONSTRAINT worksheet_organizer_worksheet_id_fkey FOREIGN KEY (worksheet_id) REFERENCES worksheet(resource_id) ON DELETE CASCADE;
CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal ON worksheet_organizer(worksheet_id, principal);

-----------------------
-- 2. issue_comment.issue_id (int → text, references issue.resource_id)
-----------------------
ALTER TABLE issue_comment DROP CONSTRAINT issue_comment_issue_id_fkey;
DROP INDEX idx_issue_comment_issue_id;

ALTER TABLE issue_comment ADD COLUMN new_issue_id text;
UPDATE issue_comment ic SET new_issue_id = i.resource_id FROM issue i WHERE ic.issue_id = i.id;
ALTER TABLE issue_comment DROP COLUMN issue_id;
ALTER TABLE issue_comment RENAME COLUMN new_issue_id TO issue_id;
ALTER TABLE issue_comment ALTER COLUMN issue_id SET NOT NULL;

ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_issue_id_fkey FOREIGN KEY (issue_id) REFERENCES issue(resource_id);
CREATE INDEX idx_issue_comment_issue_id ON issue_comment(issue_id);

-----------------------
-- 3. task_run_log.task_run_id (int → text, references task_run.resource_id)
-----------------------
ALTER TABLE task_run_log DROP CONSTRAINT task_run_log_pkey;
ALTER TABLE task_run_log DROP CONSTRAINT task_run_log_task_run_id_fkey;

ALTER TABLE task_run_log ADD COLUMN new_task_run_id text;
UPDATE task_run_log trl SET new_task_run_id = tr.resource_id FROM task_run tr WHERE trl.task_run_id = tr.id;
ALTER TABLE task_run_log DROP COLUMN task_run_id;
ALTER TABLE task_run_log RENAME COLUMN new_task_run_id TO task_run_id;
ALTER TABLE task_run_log ALTER COLUMN task_run_id SET NOT NULL;

ALTER TABLE task_run_log ADD PRIMARY KEY (task_run_id, created_at);
ALTER TABLE task_run_log ADD CONSTRAINT task_run_log_task_run_id_fkey FOREIGN KEY (task_run_id) REFERENCES task_run(resource_id);

-----------------------
-- 4. task_run.task_id (int → text, references task.resource_id)
-----------------------
DROP INDEX uk_task_run_task_id_attempt;
DROP INDEX idx_task_run_task_id;
ALTER TABLE task_run DROP CONSTRAINT task_run_task_id_fkey;

ALTER TABLE task_run ADD COLUMN new_task_id text;
UPDATE task_run tr SET new_task_id = t.resource_id FROM task t WHERE tr.task_id = t.id;
ALTER TABLE task_run DROP COLUMN task_id;
ALTER TABLE task_run RENAME COLUMN new_task_id TO task_id;
ALTER TABLE task_run ALTER COLUMN task_id SET NOT NULL;

ALTER TABLE task_run ADD CONSTRAINT task_run_task_id_fkey FOREIGN KEY (task_id) REFERENCES task(resource_id);
CREATE INDEX idx_task_run_task_id ON task_run(task_id);
CREATE UNIQUE INDEX uk_task_run_task_id_attempt ON task_run(task_id, attempt);

-----------------------
-- 5. task.plan_id (bigint → text, references plan.resource_id)
-----------------------
DROP INDEX idx_task_plan_id_environment;
ALTER TABLE task DROP CONSTRAINT task_plan_id_fkey;

ALTER TABLE task ADD COLUMN new_plan_id text;
UPDATE task t SET new_plan_id = p.resource_id FROM plan p WHERE t.plan_id = p.id;
ALTER TABLE task DROP COLUMN plan_id;
ALTER TABLE task RENAME COLUMN new_plan_id TO plan_id;
ALTER TABLE task ALTER COLUMN plan_id SET NOT NULL;

ALTER TABLE task ADD CONSTRAINT task_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan(resource_id);
CREATE INDEX idx_task_plan_id_environment ON task(plan_id, environment);

-----------------------
-- 6. issue.plan_id (bigint → text, references plan.resource_id) - NULLABLE
-----------------------
DROP INDEX idx_issue_unique_plan_id;
ALTER TABLE issue DROP CONSTRAINT issue_plan_id_fkey;

ALTER TABLE issue ADD COLUMN new_plan_id text;
UPDATE issue i SET new_plan_id = p.resource_id FROM plan p WHERE i.plan_id = p.id;
ALTER TABLE issue DROP COLUMN plan_id;
ALTER TABLE issue RENAME COLUMN new_plan_id TO plan_id;

ALTER TABLE issue ADD CONSTRAINT issue_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan(resource_id);
CREATE UNIQUE INDEX idx_issue_unique_plan_id ON issue(plan_id);

-----------------------
-- 7. plan_check_run.plan_id (bigint → text, references plan.resource_id)
-----------------------
DROP INDEX idx_plan_check_run_unique_plan_id;
DROP INDEX idx_plan_check_run_active_status;
ALTER TABLE plan_check_run DROP CONSTRAINT plan_check_run_plan_id_fkey;

ALTER TABLE plan_check_run ADD COLUMN new_plan_id text;
UPDATE plan_check_run pcr SET new_plan_id = p.resource_id FROM plan p WHERE pcr.plan_id = p.id;
ALTER TABLE plan_check_run DROP COLUMN plan_id;
ALTER TABLE plan_check_run RENAME COLUMN new_plan_id TO plan_id;
ALTER TABLE plan_check_run ALTER COLUMN plan_id SET NOT NULL;

ALTER TABLE plan_check_run ADD CONSTRAINT plan_check_run_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan(resource_id);
CREATE UNIQUE INDEX idx_plan_check_run_unique_plan_id ON plan_check_run(plan_id);
CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status IN ('AVAILABLE', 'RUNNING');

-----------------------
-- 8. plan_webhook_delivery.plan_id (bigint → text, references plan.resource_id) - plan_id IS the PK
-----------------------
ALTER TABLE plan_webhook_delivery DROP CONSTRAINT plan_webhook_delivery_pkey;
ALTER TABLE plan_webhook_delivery DROP CONSTRAINT plan_webhook_delivery_plan_id_fkey;

ALTER TABLE plan_webhook_delivery ADD COLUMN new_plan_id text;
UPDATE plan_webhook_delivery pwd SET new_plan_id = p.resource_id FROM plan p WHERE pwd.plan_id = p.id;
ALTER TABLE plan_webhook_delivery DROP COLUMN plan_id;
ALTER TABLE plan_webhook_delivery RENAME COLUMN new_plan_id TO plan_id;
ALTER TABLE plan_webhook_delivery ALTER COLUMN plan_id SET NOT NULL;

ALTER TABLE plan_webhook_delivery ADD PRIMARY KEY (plan_id);
ALTER TABLE plan_webhook_delivery ADD CONSTRAINT plan_webhook_delivery_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plan(resource_id);
