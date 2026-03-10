-- Fix workspace IAM policy resource field.
-- Currently workspace-level policies have resource = '' (empty string).
-- Set resource = 'workspaces/{workspace_id}' for SaaS multi-tenancy support.

UPDATE policy
SET resource = 'workspaces/' || (
    SELECT value->>'workspaceId'
    FROM setting
    WHERE name = 'SYSTEM'
)
WHERE resource_type = 'WORKSPACE' AND resource = '';
