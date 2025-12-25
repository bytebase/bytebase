-- Drop the non-unique index and add a unique index on plan_id.
-- This enforces the consolidated model (one record per plan) and enables UPSERT.
DROP INDEX IF EXISTS idx_plan_check_run_plan_id;
CREATE UNIQUE INDEX idx_plan_check_run_unique_plan_id ON plan_check_run(plan_id);
