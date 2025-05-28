update plan set config = replace(config::text, 'BASELINE', 'MIGRATE')::jsonb where config::text LIKE '%BASELINE%';
update task set type = 'bb.task.database.schema.update' where type = 'bb.task.database.schema.baseline';

-- Backfill sheetId in task payload from plan config
UPDATE task
SET payload = jsonb_set(
    COALESCE(payload, '{}'::jsonb),
    '{sheetId}',
    to_jsonb(CAST(SUBSTRING(spec_sheet FROM 'sheets/([0-9]+)$') AS INTEGER))
)
FROM (
    SELECT 
        t.id AS task_id,
        spec->'changeDatabaseConfig'->>'sheet' AS spec_sheet
    FROM task t
    JOIN plan p ON t.pipeline_id = p.pipeline_id
    CROSS JOIN LATERAL jsonb_array_elements(p.config->'steps') AS step
    CROSS JOIN LATERAL jsonb_array_elements(step->'specs') AS spec
    WHERE 
        t.payload IS NOT NULL
        AND t.payload->>'specId' IS NOT NULL
        AND (t.payload->>'sheetId' IS NULL OR NOT t.payload ? 'sheetId')
        AND spec->>'id' = t.payload->>'specId'
        AND spec->'changeDatabaseConfig'->>'type' = 'MIGRATE'
        AND spec->'changeDatabaseConfig'->>'sheet' IS NOT NULL
        AND SUBSTRING(spec->'changeDatabaseConfig'->>'sheet' FROM 'sheets/([0-9]+)$') IS NOT NULL
) AS plan_spec
WHERE task.id = plan_spec.task_id;