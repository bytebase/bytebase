-- Move maximum_result_size from QUERY_DATA policy to WORKSPACE_PROFILE setting
-- This migration moves the data export result size limit from the workspace-level
-- QUERY_DATA policy to the WORKSPACE_PROFILE setting (as sqlResultSize)

-- Step 1: Update or insert WORKSPACE_PROFILE setting with the maximumResultSize from QUERY_DATA policy
WITH max_result_size AS (
    SELECT (payload->>'maximumResultSize')::bigint AS size
    FROM policy
    WHERE resource_type = 'WORKSPACE'
        AND type = 'QUERY_DATA'
        AND payload ? 'maximumResultSize'
        AND payload->>'maximumResultSize' IS NOT NULL
    LIMIT 1
)
INSERT INTO setting (name, value)
SELECT
    'WORKSPACE_PROFILE',
    jsonb_build_object('sqlResultSize', max_result_size.size)
FROM max_result_size
ON CONFLICT (name)
DO UPDATE SET
    value = CASE
        WHEN setting.value IS NULL OR setting.value = '{}'::jsonb THEN
            jsonb_build_object('sqlResultSize', (SELECT size FROM max_result_size))
        ELSE
            setting.value || jsonb_build_object('sqlResultSize', (SELECT size FROM max_result_size))
    END;

-- Step 2: Remove maximumResultSize field from all QUERY_DATA policies
UPDATE policy
SET payload = payload - 'maximumResultSize'
WHERE type = 'QUERY_DATA'
    AND payload ? 'maximumResultSize';
