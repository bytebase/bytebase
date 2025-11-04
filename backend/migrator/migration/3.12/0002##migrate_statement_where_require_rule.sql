-- Migrate old statement.where.require rules to the new split rules
--
-- Context: In commit 2470780e8b (Sep 2024), the statement.where.require rule was split into two:
--   - statement.where.require.select (for SELECT statements)
--   - statement.where.require.update-delete (for UPDATE/DELETE statements)
--
-- This migration finds any remaining statement.where.require rules in review_config
-- and replaces each one with both new rule types, preserving the original level and engine.
--
-- Note: The JSONB column stores JSON marshaled by protojson.Marshal, which produces
-- camelCased keys. So we access fields as: type, level, engine (not Type, Level, Engine).

UPDATE review_config
SET payload = jsonb_set(
    payload,
    '{sqlReviewRules}',
    (
        WITH expanded_rules AS (
            SELECT
                CASE
                    -- When we find a statement.where.require rule, expand it into two rules
                    WHEN rule->>'type' = 'statement.where.require' THEN
                        jsonb_build_array(
                            jsonb_build_object(
                                'type', 'statement.where.require.select',
                                'level', rule->>'level',
                                'payload', COALESCE(rule->>'payload', ''),
                                'engine', rule->>'engine',
                                'comment', COALESCE(rule->>'comment', '')
                            ),
                            jsonb_build_object(
                                'type', 'statement.where.require.update-delete',
                                'level', rule->>'level',
                                'payload', COALESCE(rule->>'payload', ''),
                                'engine', rule->>'engine',
                                'comment', COALESCE(rule->>'comment', '')
                            )
                        )
                    -- Keep all other rules as-is, wrapped in array for consistency
                    ELSE jsonb_build_array(rule)
                END AS rules
            FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
        )
        SELECT jsonb_agg(expanded_rule)
        FROM expanded_rules, jsonb_array_elements(expanded_rules.rules) AS expanded_rule
    )
)
WHERE payload ? 'sqlReviewRules'
  -- Only process if sqlReviewRules is an array (not null, not scalar)
  AND jsonb_typeof(payload->'sqlReviewRules') = 'array'
  -- Only process if there's at least one statement.where.require rule
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
      WHERE rule->>'type' = 'statement.where.require'
  );
