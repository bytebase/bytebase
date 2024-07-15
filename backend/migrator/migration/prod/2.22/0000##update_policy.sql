UPDATE policy
SET type = 'bb.policy.iam'
WHERE type = 'bb.policy.project-iam';

UPDATE policy
SET resource_id = 0
WHERE resource_type = 'WORKSPACE';