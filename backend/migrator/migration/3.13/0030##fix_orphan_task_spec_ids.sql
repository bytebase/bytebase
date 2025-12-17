-- Migration: Fix orphan task specIds
--
-- This fixes data corruption from migration 3.7/0012##merge_changeDatabaseConfig_specs.sql
-- which merged plan specs but didn't update task.payload.specId references.
--
-- For each task with an orphan specId (one that doesn't exist in its plan's specs),
-- find the correct spec by matching the task's target against plan spec targets.

WITH orphan_tasks AS (
    -- Find tasks whose specId doesn't exist in their plan's specs
    SELECT
        t.id AS task_id,
        t.pipeline_id,
        t.instance,
        t.db_name,
        t.payload
    FROM task t
    JOIN plan p ON t.pipeline_id = p.pipeline_id
    WHERE t.payload->>'specId' IS NOT NULL
      AND NOT EXISTS (
          SELECT 1
          FROM jsonb_array_elements(p.config->'specs') AS s
          WHERE s->>'id' = t.payload->>'specId'
      )
),
correct_spec_ids AS (
    -- For each orphan task, find the correct spec by matching targets
    SELECT
        ot.task_id,
        (
            SELECT s->>'id'
            FROM plan p, jsonb_array_elements(p.config->'specs') AS s
            WHERE p.pipeline_id = ot.pipeline_id
              AND (
                  -- Match changeDatabaseConfig targets
                  EXISTS (
                      SELECT 1
                      FROM jsonb_array_elements_text(s->'changeDatabaseConfig'->'targets') AS target
                      WHERE target = 'instances/' || ot.instance || '/databases/' || ot.db_name
                  )
                  -- Match createDatabaseConfig target
                  OR s->'createDatabaseConfig'->>'target' = 'instances/' || ot.instance || '/databases/' || ot.db_name
                  -- Match exportDataConfig targets
                  OR EXISTS (
                      SELECT 1
                      FROM jsonb_array_elements_text(s->'exportDataConfig'->'targets') AS target
                      WHERE target = 'instances/' || ot.instance || '/databases/' || ot.db_name
                  )
              )
            LIMIT 1
        ) AS correct_spec_id
    FROM orphan_tasks ot
)
UPDATE task t
SET payload = jsonb_set(
    t.payload,
    '{specId}',
    to_jsonb(csi.correct_spec_id)
)
FROM correct_spec_ids csi
WHERE t.id = csi.task_id
  AND csi.correct_spec_id IS NOT NULL;
