-- Add AVAILABLE status for HA-compatible plan check scheduling.
-- Uses FOR UPDATE SKIP LOCKED pattern for atomic claiming.

-- Update status constraint to include AVAILABLE
ALTER TABLE plan_check_run
    DROP CONSTRAINT plan_check_run_status_check,
    ADD CONSTRAINT plan_check_run_status_check
        CHECK (status IN ('AVAILABLE', 'RUNNING', 'DONE', 'FAILED', 'CANCELED'));

-- Convert existing RUNNING to AVAILABLE (will be re-claimed after deployment)
UPDATE plan_check_run SET status = 'AVAILABLE' WHERE status = 'RUNNING';

-- Update index to include AVAILABLE for efficient claiming
DROP INDEX IF EXISTS idx_plan_check_run_active_status;
CREATE INDEX idx_plan_check_run_active_status ON plan_check_run(status, id) WHERE status IN ('AVAILABLE', 'RUNNING');
