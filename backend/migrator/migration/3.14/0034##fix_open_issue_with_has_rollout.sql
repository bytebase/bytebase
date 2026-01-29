-- Fix legacy data where issue is not DONE but plan.hasRollout is true.
-- This inconsistency was caused by a race condition in auto rollout creation (BYT-8790).
--
-- When a rollout is created, the expected sequence is:
--   1. Create tasks for the plan
--   2. Set plan.config.hasRollout = true
--   3. Set issue.status = DONE
-- Due to a race condition, step 3 could be skipped.

-- hasRollout is true, tasks exist, but issue is OPEN
-- → The rollout was created, so mark the issue as DONE.
UPDATE issue
SET status = 'DONE'
WHERE status = 'OPEN'
  AND type = 'DATABASE_CHANGE'
  AND plan_id IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM plan
    WHERE plan.id = issue.plan_id
      AND plan.config->>'hasRollout' = 'true'
  );
