-- Migration: Remove execution events from issue_comment
--
-- This migration:
-- 1. Deletes stageEnd comments (derivable from task statuses)
-- 2. Deletes taskPriorBackup comments (tracked in task_run_log)
-- 3. Deletes taskUpdate comments with toStatus (execution status tracked in task_run)
-- 4. Migrates taskUpdate comments with sheet changes to planSpecUpdate

-- Step 1: Delete stageEnd comments
DELETE FROM issue_comment WHERE payload ? 'stageEnd';

-- Step 2: Delete taskPriorBackup comments
DELETE FROM issue_comment WHERE payload ? 'taskPriorBackup';

-- Step 3: Delete taskUpdate comments that only have status changes (no sheet changes)
DELETE FROM issue_comment
WHERE payload ? 'taskUpdate'
  AND (payload->'taskUpdate') ? 'toStatus'
  AND NOT ((payload->'taskUpdate') ? 'fromSheet' OR (payload->'taskUpdate') ? 'toSheet');

-- Step 4: Migrate taskUpdate comments with sheet changes to planSpecUpdate
-- The task name format is: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task_id}
-- The spec format is: projects/{project}/plans/{plan_id}/specs/{spec_id}
UPDATE issue_comment ic
SET payload = jsonb_build_object(
    'planSpecUpdate', jsonb_build_object(
        'spec', 'projects/' || i.project || '/plans/' || i.plan_id || '/specs/' || COALESCE(t.payload->>'specId', ''),
        'fromSheet', ic.payload->'taskUpdate'->>'fromSheet',
        'toSheet', ic.payload->'taskUpdate'->>'toSheet'
    )
)
FROM issue i, task t
WHERE ic.payload ? 'taskUpdate'
  AND ((ic.payload->'taskUpdate') ? 'fromSheet' OR (ic.payload->'taskUpdate') ? 'toSheet')
  AND ic.issue_id = i.id
  AND i.plan_id IS NOT NULL
  AND t.id = (
      SELECT CAST(
          split_part(
              ic.payload->'taskUpdate'->'tasks'->>0,
              '/tasks/',
              2
          ) AS INTEGER
      )
  );

-- Step 5: Delete any remaining taskUpdate comments that couldn't be migrated
DELETE FROM issue_comment
WHERE payload ? 'taskUpdate';
