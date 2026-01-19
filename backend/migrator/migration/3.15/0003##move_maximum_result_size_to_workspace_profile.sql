-- Move maximum_result_size from QUERY_DATA policy to WORKSPACE_PROFILE setting
-- This migration moves the data export result size limit from the workspace-level
-- QUERY_DATA policy to the WORKSPACE_PROFILE setting (as dataExportResultSize)

-- Step 1: Update or insert WORKSPACE_PROFILE setting with the maximumResultSize from QUERY_DATA policy
INSERT INTO setting (name, value)
SELECT
    'WORKSPACE_PROFILE',
    jsonb_build_object('dataExportResultSize', (p.payload->>'maximumResultSize')::bigint)
FROM policy p
WHERE p.resource_type = 'WORKSPACE'
    AND p.type = 'QUERY_DATA'
    AND p.payload ? 'maximumResultSize'
    AND p.payload->>'maximumResultSize' IS NOT NULL
ON CONFLICT (name)
DO UPDATE SET
    value = CASE
        WHEN setting.value IS NULL OR setting.value = '{}'::jsonb THEN
            jsonb_build_object('dataExportResultSize', (
                SELECT (payload->>'maximumResultSize')::bigint
                FROM policy
                WHERE resource_type = 'WORKSPACE'
                    AND type = 'QUERY_DATA'
                    AND payload ? 'maximumResultSize'
                    AND payload->>'maximumResultSize' IS NOT NULL
                LIMIT 1
            ))
        ELSE
            setting.value || jsonb_build_object('dataExportResultSize', (
                SELECT (payload->>'maximumResultSize')::bigint
                FROM policy
                WHERE resource_type = 'WORKSPACE'
                    AND type = 'QUERY_DATA'
                    AND payload ? 'maximumResultSize'
                    AND payload->>'maximumResultSize' IS NOT NULL
                LIMIT 1
            ))
    END;

-- Step 2: Remove maximumResultSize field from all QUERY_DATA policies
UPDATE policy
SET payload = payload - 'maximumResultSize'
WHERE type = 'QUERY_DATA'
    AND payload ? 'maximumResultSize';
