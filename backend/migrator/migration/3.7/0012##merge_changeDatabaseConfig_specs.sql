-- Query to merge plan specs with the same sheet in changeDatabaseConfig
-- This will group specs by sheet and merge their targets arrays

WITH specs_with_sheet AS (
    -- Extract specs that have changeDatabaseConfig with a sheet
    SELECT 
        p.id AS plan_id,
        p.project,
        p.name,
        spec_index,
        spec->>'id' AS spec_id,
        spec->'changeDatabaseConfig'->>'sheet' AS sheet,
        spec->'changeDatabaseConfig'->'targets' AS targets,
        spec->'changeDatabaseConfig' AS change_config
    FROM 
        plan p,
        jsonb_array_elements(p.config->'specs') WITH ORDINALITY AS specs(spec, spec_index)
    WHERE 
        spec->'changeDatabaseConfig' IS NOT NULL
        AND spec->'changeDatabaseConfig'->>'sheet' IS NOT NULL
),
merged_specs AS (
    -- Group by plan_id and sheet, merge all targets arrays
    SELECT 
        plan_id,
        project,
        sheet,
        MIN(spec_index) AS min_index,  -- Keep the lowest index for ordering
        JSONB_BUILD_OBJECT(
            'id', (ARRAY_AGG(spec_id ORDER BY spec_index))[1],  -- Keep the first spec's id
            'changeDatabaseConfig', 
                JSONB_BUILD_OBJECT(
                    'targets', JSONB_AGG(DISTINCT target ORDER BY target),  -- Merge and deduplicate targets
                    'sheet', sheet,
                    'type', (ARRAY_AGG(change_config->>'type' ORDER BY spec_index))[1]  -- Take first spec's type
                )
                || CASE 
                    WHEN (ARRAY_AGG(change_config->>'release' ORDER BY spec_index))[1] IS NOT NULL 
                    THEN JSONB_BUILD_OBJECT('release', (ARRAY_AGG(change_config->>'release' ORDER BY spec_index))[1])
                    ELSE '{}'::jsonb
                END
                || CASE 
                    WHEN (ARRAY_AGG(change_config->'ghostFlags' ORDER BY spec_index))[1] IS NOT NULL 
                         AND (ARRAY_AGG(change_config->'ghostFlags' ORDER BY spec_index))[1] != 'null'::jsonb
                    THEN JSONB_BUILD_OBJECT('ghostFlags', (ARRAY_AGG(change_config->'ghostFlags' ORDER BY spec_index))[1])
                    ELSE '{}'::jsonb
                END
                || CASE 
                    WHEN (ARRAY_AGG(change_config->'preUpdateBackupDetail' ORDER BY spec_index))[1] IS NOT NULL 
                         AND (ARRAY_AGG(change_config->'preUpdateBackupDetail' ORDER BY spec_index))[1] != 'null'::jsonb
                    THEN JSONB_BUILD_OBJECT('preUpdateBackupDetail', (ARRAY_AGG(change_config->'preUpdateBackupDetail' ORDER BY spec_index))[1])
                    ELSE '{}'::jsonb
                END
        ) AS merged_spec
    FROM 
        specs_with_sheet,
        jsonb_array_elements_text(targets) AS target
    GROUP BY 
        plan_id, project, sheet
),
other_specs AS (
    -- Get all specs that are NOT changeDatabaseConfig or don't have a sheet
    SELECT 
        p.id AS plan_id,
        spec_index,
        spec
    FROM 
        plan p,
        jsonb_array_elements(p.config->'specs') WITH ORDINALITY AS specs(spec, spec_index)
    WHERE 
        spec->'changeDatabaseConfig' IS NULL
        OR spec->'changeDatabaseConfig'->>'sheet' IS NULL
),
new_specs_array AS (
    -- Combine merged specs and other specs, maintaining order
    SELECT 
        plan_id,
        JSONB_AGG(
            spec ORDER BY 
            CASE 
                WHEN source = 'merged' THEN min_index 
                ELSE spec_index 
            END
        ) AS new_specs
    FROM (
        -- Merged specs
        SELECT 
            plan_id,
            merged_spec AS spec,
            min_index,
            NULL::bigint AS spec_index,
            'merged' AS source
        FROM merged_specs
        
        UNION ALL
        
        -- Other specs
        SELECT 
            plan_id,
            spec,
            NULL::real AS min_index,
            spec_index,
            'other' AS source
        FROM other_specs
    ) combined_specs
    GROUP BY plan_id
)
-- Update the plan table with merged specs
UPDATE plan
SET 
    config = JSONB_SET(
        config,
        '{specs}',
        COALESCE(nsa.new_specs, '[]'::jsonb)
    ),
    updated_at = NOW()
FROM new_specs_array nsa
WHERE 
    plan.id = nsa.plan_id
    -- Only update plans that have duplicate sheets
    AND EXISTS (
        SELECT 1
        FROM specs_with_sheet sws
        WHERE sws.plan_id = plan.id
        GROUP BY sws.plan_id, sws.sheet
        HAVING COUNT(*) > 1
    );
