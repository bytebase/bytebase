-- Remove invalid column.set-default-for-not-null rules for Oracle
--
-- Context: The column.set-default-for-not-null advisor was never implemented for Oracle.
-- It only exists for MySQL, TiDB, OceanBase, and MariaDB. However, the frontend SQL Review
-- Policy templates incorrectly included this rule for Oracle.
--
-- After commit e3861d41 (Sep 2025), unknown advisors are no longer silently ignored,
-- causing errors when these invalid rules are encountered.
--
-- This migration removes any column.set-default-for-not-null rules with engine=ORACLE
-- from existing review_config entries.
--
-- Note: The JSONB column stores JSON marshaled by protojson.Marshal, which produces
-- camelCased keys. So we access fields as: type, level, engine (not Type, Level, Engine).

UPDATE review_config
SET payload = jsonb_set(
    payload,
    '{sqlReviewRules}',
    (
        SELECT jsonb_agg(rule)
        FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
        WHERE NOT (
            rule->>'type' = 'column.set-default-for-not-null'
            AND rule->>'engine' = 'ORACLE'
        )
    )
)
WHERE payload ? 'sqlReviewRules'
  -- Only process if sqlReviewRules is an array (not null, not scalar)
  AND jsonb_typeof(payload->'sqlReviewRules') = 'array'
  -- Only process if there's at least one invalid rule to remove
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
      WHERE rule->>'type' = 'column.set-default-for-not-null'
        AND rule->>'engine' = 'ORACLE'
  );
