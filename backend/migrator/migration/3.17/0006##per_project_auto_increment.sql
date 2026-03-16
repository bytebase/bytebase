-- Remove global sequence defaults, drop sequences, unify id columns to bigint.
-- Drop unused resource_id columns from plan, task, task_run, issue.
-- NOTE: DROP DEFAULT / DROP COLUMN IF EXISTS / DROP INDEX IF EXISTS are all idempotent.

-- Drop unused resource_id columns and their unique indexes.
DROP INDEX IF EXISTS idx_plan_unique_resource_id;
DROP INDEX IF EXISTS idx_task_unique_resource_id;
DROP INDEX IF EXISTS idx_task_run_unique_resource_id;
DROP INDEX IF EXISTS idx_issue_unique_resource_id;
ALTER TABLE plan DROP COLUMN IF EXISTS resource_id;
ALTER TABLE task DROP COLUMN IF EXISTS resource_id;
ALTER TABLE task_run DROP COLUMN IF EXISTS resource_id;
ALTER TABLE issue DROP COLUMN IF EXISTS resource_id;

-- Remove global sequence defaults and unify id columns to bigint.
ALTER TABLE plan ALTER COLUMN id DROP DEFAULT;
ALTER TABLE task ALTER COLUMN id DROP DEFAULT;
ALTER TABLE task ALTER COLUMN id SET DATA TYPE bigint;
ALTER TABLE task_run ALTER COLUMN id DROP DEFAULT;
ALTER TABLE task_run ALTER COLUMN id SET DATA TYPE bigint;
ALTER TABLE plan_check_run ALTER COLUMN id DROP DEFAULT;
ALTER TABLE plan_check_run ALTER COLUMN id SET DATA TYPE bigint;
ALTER TABLE issue ALTER COLUMN id DROP DEFAULT;
ALTER TABLE issue ALTER COLUMN id SET DATA TYPE bigint;
ALTER TABLE worksheet ALTER COLUMN id DROP DEFAULT;
ALTER TABLE worksheet ALTER COLUMN id SET DATA TYPE bigint;

DROP SEQUENCE IF EXISTS plan_id_seq;
DROP SEQUENCE IF EXISTS task_id_seq;
DROP SEQUENCE IF EXISTS task_run_id_seq;
DROP SEQUENCE IF EXISTS plan_check_run_id_seq;
DROP SEQUENCE IF EXISTS issue_id_seq;
DROP SEQUENCE IF EXISTS worksheet_id_seq;
