-- Migrate RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW policy from workspace to project level
-- This will copy the workspace-level policy to all projects that don't already have this policy

-- First, insert the policy for all projects that don't already have it
INSERT INTO policy (resource_type, resource, type, payload, enforce, inherit_from_parent)
SELECT
    'PROJECT' as resource_type,
    CONCAT('projects/', p.resource_id) as resource,
    'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW' as type,
    wp.payload,
    wp.enforce,
    false as inherit_from_parent
FROM project p
CROSS JOIN (
    SELECT payload, enforce
    FROM policy
    WHERE resource_type = 'WORKSPACE'
    AND resource = ''
    AND type = 'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW'
    LIMIT 1
) wp
WHERE p.deleted = false
AND NOT EXISTS (
    SELECT 1 FROM policy pp
    WHERE pp.resource_type = 'PROJECT'
    AND pp.resource = CONCAT('projects/', p.resource_id)
    AND pp.type = 'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW'
);

-- Delete the workspace-level policy
DELETE FROM policy
WHERE resource_type = 'WORKSPACE'
AND resource = ''
AND type = 'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW';