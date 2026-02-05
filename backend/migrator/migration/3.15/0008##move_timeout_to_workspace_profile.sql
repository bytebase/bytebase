-- Move timeout from QUERY_DATA policy to WORKSPACE_PROFILE setting
-- This migration moves the query timeout duration from the workspace-level
-- QUERY_DATA policy to the WORKSPACE_PROFILE setting (as queryTimeout)

-- Step 1: Update or insert WORKSPACE_PROFILE setting with the timeout from QUERY_DATA policy
WITH query_timeout AS (
    SELECT payload->>'timeout' AS timeout
    FROM policy
    WHERE resource_type = 'WORKSPACE'
        AND type = 'QUERY_DATA'
        AND payload ? 'timeout'
        AND payload->>'timeout' IS NOT NULL
    LIMIT 1
)
INSERT INTO setting (name, value)
SELECT
    'WORKSPACE_PROFILE',
    jsonb_build_object('queryTimeout', query_timeout.timeout)
FROM query_timeout
ON CONFLICT (name)
DO UPDATE SET
    value = CASE
        WHEN setting.value IS NULL OR setting.value = '{}'::jsonb THEN
            jsonb_build_object('queryTimeout', (SELECT timeout FROM query_timeout))
        ELSE
            setting.value || jsonb_build_object('queryTimeout', (SELECT timeout FROM query_timeout))
    END;

-- Step 2: Remove timeout field from all QUERY_DATA policies
UPDATE policy
SET payload = payload - 'timeout'
WHERE type = 'QUERY_DATA'
    AND payload ? 'timeout';
