-- Add default checkers configuration to existing rollout policies
-- Set required_issue_approval to true and plan_check_enforcement to ERROR_ONLY

UPDATE policy
SET payload = jsonb_set(
    jsonb_set(
        payload,
        '{checkers}',
        jsonb_build_object(
            'requiredIssueApproval', true,
            'requiredStatusChecks', jsonb_build_object(
                'planCheckEnforcement', 'ERROR_ONLY'
            )
        )
    ),
    '{checkers}',
    payload->'checkers' || jsonb_build_object(
        'requiredIssueApproval', true,
        'requiredStatusChecks', jsonb_build_object(
            'planCheckEnforcement', 'ERROR_ONLY'
        )
    )
)
WHERE type = 'ROLLOUT'
AND (payload->'checkers' IS NULL OR payload->'checkers' = '{}');