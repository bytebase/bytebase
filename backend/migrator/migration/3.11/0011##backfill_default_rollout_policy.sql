-- Backfill default rollout policies for environments that don't have one
-- This ensures all environments have a rollout policy with default checkers

WITH environment_ids AS (
    SELECT jsonb_array_elements(value::jsonb->'environments')->>'id' AS env_id
    FROM setting
    WHERE name = 'ENVIRONMENT'
)
INSERT INTO policy (resource_type, resource, type, payload, inherit_from_parent, enforce, updated_at)
SELECT
    'ENVIRONMENT',
    'environments/' || env_id,
    'ROLLOUT',
    jsonb_build_object(
        'checkers', jsonb_build_object(
            'requiredIssueApproval', true,
            'requiredStatusChecks', jsonb_build_object(
                'planCheckEnforcement', 'ERROR_ONLY'
            )
        )
    ),
    true,
    true,
    now()
FROM environment_ids
WHERE NOT EXISTS (
    SELECT 1
    FROM policy p
    WHERE p.resource_type = 'ENVIRONMENT'
    AND p.resource = 'environments/' || env_id
    AND p.type = 'ROLLOUT'
);
