-- Replace non-unique index with unique index to enable UPSERT.
DROP INDEX IF EXISTS idx_plan_check_run_plan_id;
CREATE UNIQUE INDEX idx_plan_check_run_unique_plan_id ON plan_check_run(plan_id);
