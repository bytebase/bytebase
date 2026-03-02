-- Fix legacy issues that were incorrectly moved to DONE by 3.14/0034.
--
-- Migration 3.14/0034 transitioned all OPEN issues with hasRollout=true to DONE.
-- This was correct for issues that were already approved (or didn't need approval),
-- but incorrect for issues still pending approval — they were never executed
-- (no task runs) and should remain open so users can complete the approval flow.
--
-- Valid DONE state (keep as-is):
--   - No approval needed (no approvalTemplate)
--   - Approval skipped (approvalTemplate is null)
--   - All roles approved (every approver has APPROVED status)
--
-- Invalid state (cleanup):
--   - Has approvalTemplate with roles AND approval not complete
--   - AND no task runs (nothing was executed)
--
-- This migration cleans up invalid states:
--   1. Record affected plan IDs
--   2. Delete orphaned tasks (no task runs, so nothing to preserve)
--   3. Reopen the issue (DONE → OPEN), preserving existing approval state
--   4. Clear hasRollout on the plan (rollout will be recreated after approval)
--
-- NOTE: CASE is used to guard jsonb_array_length because PostgreSQL does not
-- guarantee left-to-right evaluation of AND conditions. Without CASE, the
-- optimizer could evaluate jsonb_array_length before jsonb_typeof, crashing
-- on non-array values.

-- Step 1: Record affected plan IDs once, then reuse for all modifications.
CREATE TEMP TABLE affected_plan_id (
    plan_id BIGINT PRIMARY KEY
);

INSERT INTO affected_plan_id (plan_id)
SELECT plan.id
FROM issue
JOIN plan ON plan.id = issue.plan_id
WHERE issue.status = 'DONE'
  AND issue.type = 'DATABASE_CHANGE'
  AND issue.plan_id IS NOT NULL
  AND plan.deleted = false
  AND plan.config->>'hasRollout' = 'true'
  -- Has approval template with non-empty roles (approval is required).
  AND CASE WHEN jsonb_typeof(issue.payload #> '{approval,approvalTemplate,flow,roles}') = 'array'
           THEN jsonb_array_length(issue.payload #> '{approval,approvalTemplate,flow,roles}') > 0
           ELSE false END
  -- Approval not complete: has pending steps OR has a non-APPROVED approver.
  -- Issues where all roles are filled with APPROVED status are truly approved
  -- and should stay DONE.
  AND (
      CASE WHEN jsonb_typeof(issue.payload #> '{approval,approvers}') = 'array'
           THEN jsonb_array_length(issue.payload #> '{approval,approvers}')
           ELSE 0 END
        < CASE WHEN jsonb_typeof(issue.payload #> '{approval,approvalTemplate,flow,roles}') = 'array'
               THEN jsonb_array_length(issue.payload #> '{approval,approvalTemplate,flow,roles}')
               ELSE 0 END
      OR EXISTS (
          SELECT 1
          FROM jsonb_array_elements(
              CASE WHEN jsonb_typeof(issue.payload #> '{approval,approvers}') = 'array'
                   THEN issue.payload #> '{approval,approvers}'
                   ELSE '[]'::jsonb END
          ) AS a
          WHERE COALESCE(a->>'status', '') != 'APPROVED'
      )
  )
  -- No task runs means nothing was executed.
  AND NOT EXISTS (
      SELECT 1
      FROM task t
      JOIN task_run tr ON tr.task_id = t.id
      WHERE t.plan_id = plan.id
  );

-- Step 2: Delete orphaned tasks (no task runs exist, so nothing to preserve).
DELETE FROM task
WHERE plan_id IN (SELECT plan_id FROM affected_plan_id);

-- Step 3: Reopen the issues. The existing approval state (approvalTemplate,
-- approvers, approvalFindingDone) is preserved so the Approve button works
-- immediately without needing the approval runner to re-process.
UPDATE issue
SET status = 'OPEN'
WHERE plan_id IN (SELECT plan_id FROM affected_plan_id);

-- Step 4: Clear hasRollout on the affected plans. The rollout will be
-- recreated after the user completes the approval flow.
UPDATE plan
SET config = jsonb_set(config, '{hasRollout}', 'false'::jsonb)
WHERE id IN (SELECT plan_id FROM affected_plan_id);

DROP TABLE affected_plan_id;
