-- Drop the CHECK constraint first
ALTER TABLE policy DROP CONSTRAINT IF EXISTS policy_type_check;

-- Update policy type values to match proto enum names
UPDATE policy SET type = CASE
    WHEN type = 'bb.policy.rollout' THEN 'ROLLOUT'
    WHEN type = 'bb.policy.masking-exception' THEN 'MASKING_EXCEPTION'
    WHEN type = 'bb.policy.disable-copy-data' THEN 'DISABLE_COPY_DATA'
    WHEN type = 'bb.policy.export-data' THEN 'EXPORT_DATA'
    WHEN type = 'bb.policy.query-data' THEN 'QUERY_DATA'
    WHEN type = 'bb.policy.masking-rule' THEN 'MASKING_RULE'
    WHEN type = 'bb.policy.restrict-issue-creation-for-sql-review' THEN 'RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW'
    WHEN type = 'bb.policy.iam' THEN 'IAM'
    WHEN type = 'bb.policy.tag' THEN 'TAG'
    WHEN type = 'bb.policy.data-source-query' THEN 'DATA_SOURCE_QUERY'
    ELSE type
END;