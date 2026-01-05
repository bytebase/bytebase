-- Drop config and payload columns from plan_check_run
ALTER TABLE plan_check_run DROP COLUMN config;
ALTER TABLE plan_check_run DROP COLUMN payload;
