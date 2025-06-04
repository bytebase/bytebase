-- Drop the existing check constraint
ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_type_check;

-- Backfill issue type from string values to enum values
UPDATE issue
SET type = CASE
    WHEN type = 'bb.issue.grant.request' THEN 'GRANT_REQUEST'
    WHEN type = 'bb.issue.database.general' THEN 'DATABASE_CHANGE'
    WHEN type = 'bb.issue.database.data-export' THEN 'DATABASE_EXPORT'
END
WHERE type IS NOT NULL;
