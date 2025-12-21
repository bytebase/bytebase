-- Insert a FAILED task run for tasks that are currently blocked by a failing 'Database Connect' check.
-- This prevents the auto-rollout scheduler from automatically picking them up once the check is deleted.
INSERT INTO task_run (
    creator,
    task_id,
    attempt,
    status,
    result
)
SELECT DISTINCT
    'support@bytebase.com',
    t.id,
    0,
    'FAILED',
    '{"detail": "Automatically failed by migration because the blocking ''Database Connect'' plan check was removed. Please verify database connection manually and retry."}'::jsonb
FROM task t
JOIN pipeline p ON p.id = t.pipeline_id
JOIN plan pl ON pl.pipeline_id = p.id
JOIN plan_check_run pcr ON pcr.plan_id = pl.id
WHERE
    pcr.type = 'bb.plan-check.database.connect'
    AND (
        pcr.status = 'FAILED'
        OR EXISTS (
            SELECT 1
            FROM jsonb_array_elements(pcr.result->'results') AS elem
            WHERE elem->>'status' = 'ERROR'
        )
    )
    AND NOT EXISTS (SELECT 1 FROM task_run tr WHERE tr.task_id = t.id);

-- Remove ALL PlanCheckDatabaseConnect and fake advise plan check runs
DELETE FROM plan_check_run WHERE type IN ('bb.plan-check.database.connect', 'bb.plan-check.database.statement.fake-advise');