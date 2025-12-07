-- Remove DISABLED rules from SQL review configurations.
--
-- Context: SQLReviewRule_Level enum previously had DISABLED as an option.
-- This has been removed - rules should either be in the policy (ERROR/WARNING) or absent.
--
-- This migration filters out any rules with level = "DISABLED" from the sqlReviewRules array
-- in the review_config.payload JSONB column.
--
-- Note: The JSONB column stores JSON marshaled by protojson.Marshal, which uses string enum
-- names (not integers). So level values are: "ERROR", "WARNING", "DISABLED", etc.
--
-- Structure: payload.sqlReviewRules[].level where level can be:
--   "LEVEL_UNSPECIFIED"
--   "ERROR"
--   "WARNING"
--   "DISABLED" (to be removed)

UPDATE review_config
SET payload = jsonb_set(
    payload,
    '{sqlReviewRules}',
    COALESCE(
        (
            SELECT jsonb_agg(rule)
            FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
            WHERE (rule->>'level') != 'DISABLED'
        ),
        '[]'::jsonb
    )
)
WHERE payload ? 'sqlReviewRules'
  AND EXISTS (
      SELECT 1
      FROM jsonb_array_elements(payload->'sqlReviewRules') AS rule
      WHERE (rule->>'level') = 'DISABLED'
  );
