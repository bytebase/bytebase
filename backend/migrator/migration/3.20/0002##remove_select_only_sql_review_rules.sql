-- Remove 7 SELECT-only SQL review rules from review configurations.
--
-- Context: These rules only triggered on SELECT statements and served no purpose
-- in the DDL/DML change review workflow. Their proto enum values have been reserved
-- in PR #20685 (BYT-9784), so existing payloads referencing these type names must
-- be cleaned up to avoid protojson unmarshal failures.
--
-- Removed rules:
--   STATEMENT_SELECT_FULL_TABLE_SCAN
--   STATEMENT_QUERY_MINIMUM_PLAN_LEVEL
--   STATEMENT_DISALLOW_USING_FILESORT
--   STATEMENT_DISALLOW_USING_TEMPORARY
--   STATEMENT_MAXIMUM_JOIN_TABLE_COUNT
--   STATEMENT_JOIN_STRICT_COLUMN_ATTRS
--   STATEMENT_MAXIMUM_LIMIT_VALUE

UPDATE review_config
SET payload = jsonb_set(
    payload,
    '{sqlReviewRules}',
    COALESCE(
        (
            SELECT jsonb_agg(rule)
            FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
            WHERE (rule->>'type') NOT IN (
                'STATEMENT_SELECT_FULL_TABLE_SCAN',
                'STATEMENT_QUERY_MINIMUM_PLAN_LEVEL',
                'STATEMENT_DISALLOW_USING_FILESORT',
                'STATEMENT_DISALLOW_USING_TEMPORARY',
                'STATEMENT_MAXIMUM_JOIN_TABLE_COUNT',
                'STATEMENT_JOIN_STRICT_COLUMN_ATTRS',
                'STATEMENT_MAXIMUM_LIMIT_VALUE'
            )
        ),
        '[]'::jsonb
    )
)
WHERE payload ? 'sqlReviewRules'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
      WHERE (rule->>'type') IN (
          'STATEMENT_SELECT_FULL_TABLE_SCAN',
          'STATEMENT_QUERY_MINIMUM_PLAN_LEVEL',
          'STATEMENT_DISALLOW_USING_FILESORT',
          'STATEMENT_DISALLOW_USING_TEMPORARY',
          'STATEMENT_MAXIMUM_JOIN_TABLE_COUNT',
          'STATEMENT_JOIN_STRICT_COLUMN_ATTRS',
          'STATEMENT_MAXIMUM_LIMIT_VALUE'
      )
  );
