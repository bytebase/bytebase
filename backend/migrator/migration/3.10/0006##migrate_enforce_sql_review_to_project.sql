-- Migrate RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW policy to project setting.enforce_sql_review field
-- This migration moves the policy data from the policy table to the project setting JSONB field

-- Update project setting to set enforce_sql_review based on existing RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW policies
UPDATE project p
SET setting = jsonb_set(
    COALESCE(setting, '{}'::jsonb),
    '{enforceSqlReview}',
    'true'::jsonb,
    true
)
WHERE EXISTS (
    SELECT 1
    FROM policy pol
    WHERE pol.resource_type = 'PROJECT'
    AND pol.resource = CONCAT('projects/', p.resource_id)
    AND pol.type = 'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW'
    AND pol.payload::jsonb->>'disallow' = 'true'
    AND pol.enforce = true
);

-- Clean up: Delete the deprecated RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW policies
DELETE FROM policy
WHERE type = 'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW';