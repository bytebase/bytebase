WITH plan_projects AS (
    SELECT DISTINCT project_id
    FROM plan
    WHERE EXISTS (
        SELECT 1
        FROM jsonb_array_elements(config->'steps') AS steps,
             jsonb_array_elements(steps->'specs') AS specs
        WHERE specs->'changeDatabaseConfig'->>'target' LIKE '%/deploymentConfigs/%'
    )
),
projects_without_all_databases AS (
    SELECT DISTINCT dbg.project_id
    FROM db_group dbg
    JOIN plan_projects pp ON dbg.project_id = pp.project_id
    WHERE dbg.project_id NOT IN (
        SELECT project_id
        FROM db_group
        WHERE resource_id = 'all-databases'
    )
)
INSERT INTO db_group (creator_id, updater_id, project_id, resource_id, placeholder, expression, payload)
SELECT 
    p.creator_id,
    p.updater_id,
    p.id,
    'all-databases' AS resource_id,
    'all-databases' AS placeholder,
    '{"expression": "true"}'::jsonb AS expression,
    '{"multitenancy": true}'::jsonb AS payload
FROM 
    project p
WHERE 
    p.id IN (SELECT project_id FROM projects_without_all_databases);

WITH plan_to_update AS (
    SELECT id, config
    FROM plan
    WHERE EXISTS (
        SELECT 1
        FROM jsonb_array_elements(config->'steps') AS steps,
             jsonb_array_elements(steps->'specs') AS specs
        WHERE specs->'changeDatabaseConfig'->>'target' LIKE '%/deploymentConfigs/%'
    )
)
UPDATE plan AS p
SET config = jsonb_set(
    p.config,
    '{steps}',
    (
        SELECT jsonb_agg(
            jsonb_set(
                step,
                '{specs}',
                (
                    SELECT jsonb_agg(
                        jsonb_set(
                            spec,
                            '{changeDatabaseConfig,target}',
                            ('"' || regexp_replace(
                                spec->'changeDatabaseConfig'->>'target',
                                '/deploymentConfigs/.*$',
                                '/databaseGroups/all-databases'
                            ) || '"')::jsonb
                        )
                    )
                    FROM jsonb_array_elements(step->'specs') AS spec
                )
            )
        )
        FROM jsonb_array_elements(p.config->'steps') AS step
    )
)
FROM plan_to_update AS pu
WHERE p.id = pu.id;
