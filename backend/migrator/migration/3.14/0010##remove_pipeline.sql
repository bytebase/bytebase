ALTER TABLE task ADD COLUMN plan_id bigint REFERENCES plan(id);

UPDATE task SET plan_id = plan.id
FROM pipeline
JOIN plan ON plan.pipeline_id = pipeline.id
WHERE task.pipeline_id = pipeline.id;

-- Remove any orphaned tasks that don't belong to a plan (i.e. not linked to a pipeline).
-- This is required to satisfy the NOT NULL constraint on plan_id.
DELETE FROM task_run_log USING task_run, task
WHERE task_run_log.task_run_id = task_run.id AND task_run.task_id = task.id AND task.plan_id IS NULL;

DELETE FROM task_run USING task
WHERE task_run.task_id = task.id AND task.plan_id IS NULL;

DELETE FROM task WHERE plan_id IS NULL;

ALTER TABLE task ALTER COLUMN plan_id SET NOT NULL;

-- Backfill taskRun in changelog payload to use plan ID and new resource naming scheme
UPDATE changelog
SET payload = jsonb_set(
    payload,
    '{taskRun}',
    to_jsonb(
        replace(
            payload->>'taskRun',
            '/rollouts/' || pipeline.id || '/',
            '/plans/' || plan.id || '/rollout/'
        )
    )
)
FROM plan, pipeline
WHERE plan.pipeline_id = pipeline.id
AND changelog.payload->>'taskRun' LIKE '%/rollouts/' || pipeline.id || '/%';

-- Backfill taskRun in revision payload to use plan ID and new resource naming scheme
UPDATE revision
SET payload = jsonb_set(
    payload,
    '{taskRun}',
    to_jsonb(
        replace(
            payload->>'taskRun',
            '/rollouts/' || pipeline.id || '/',
            '/plans/' || plan.id || '/rollout/'
        )
    )
)
FROM plan, pipeline
WHERE plan.pipeline_id = pipeline.id
AND revision.payload->>'taskRun' LIKE '%/rollouts/' || pipeline.id || '/%';

DROP INDEX IF EXISTS idx_task_pipeline_id_environment;
ALTER TABLE task DROP COLUMN pipeline_id;

DROP INDEX IF EXISTS idx_plan_unique_pipeline_id;
ALTER TABLE plan DROP COLUMN pipeline_id;
DROP TABLE pipeline;

CREATE INDEX idx_task_plan_id_environment ON task(plan_id, environment);

-- Backfill hasRollout in plan config based on existing tasks
UPDATE plan
SET config = jsonb_set(COALESCE(config, '{}'::jsonb), '{hasRollout}', 'true'::jsonb)
WHERE EXISTS (
    SELECT 1
    FROM task
    WHERE task.plan_id = plan.id
);
