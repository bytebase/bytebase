ALTER TABLE policy DISABLE TRIGGER update_policy_updated_ts;

INSERT INTO policy (
    row_status,
    creator_id,
    created_ts,
    updater_id,
    updated_ts,
    type,
    payload,
    resource_type,
    resource_id,
    inherit_from_parent
) SELECT
    row_status,
    creator_id,
    created_ts,
    updater_id,
    updated_ts,
    'bb.policy.rollout',
    CASE
        WHEN payload @> '{"value": "MANUAL_APPROVAL_NEVER"}' THEN '{"automatic": true}'::jsonb
        WHEN payload @> '{
            "value": "MANUAL_APPROVAL_ALWAYS",
            "assigneeGroupList": [{"value": "PROJECT_OWNER"}]
        }' THEN '{
            "projectRoles": ["roles/OWNER"]
        }'::jsonb
        WHEN payload @> '{
            "value": "MANUAL_APPROVAL_ALWAYS",
            "assigneeGroupList": [{"value": "WORKSPACE_OWNER_OR_DBA"}]
        }' THEN '{
            "workspaceRoles": ["roles/OWNER", "roles/DBA"]
        }'::jsonb
        ELSE '{"automatic": true}'::jsonb
    END,
    resource_type,
    resource_id,
    inherit_from_parent
FROM policy
WHERE type = 'bb.policy.pipeline-approval';

ALTER TABLE policy ENABLE TRIGGER update_policy_updated_ts;
