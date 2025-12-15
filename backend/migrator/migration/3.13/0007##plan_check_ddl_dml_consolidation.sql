-- Consolidate ChangeDatabaseType enum to EnableSDL boolean in plan_check_run configs.
-- DDL(1), DML(2), DDL_GHOST(4) -> enableSdl: false (or absent)
-- SDL(3) -> enableSdl: true

-- Set enableSdl = true for SDL configs
UPDATE plan_check_run
SET config = jsonb_set(config, '{enableSdl}', 'true')
WHERE config->>'changeDatabaseType' = 'SDL';

-- Remove the deprecated changeDatabaseType field from all configs
UPDATE plan_check_run
SET config = config - 'changeDatabaseType'
WHERE config ? 'changeDatabaseType';

-- Remove disallow-mix rules and built-in rules from review_config if they exist.
-- Built-in rules (like prior-backup-check) are automatically injected by GetBuiltinRules()
-- and should never be stored in the database.
UPDATE review_config
SET payload = jsonb_set(
  payload,
  '{sqlReviewRules}',
  COALESCE(
    (SELECT jsonb_agg(r)
     FROM jsonb_array_elements(payload->'sqlReviewRules') r
     WHERE r->>'type' NOT IN (
       'statement.disallow-mix-in-ddl',
       'statement.disallow-mix-in-dml',
       'statement.prior-backup-check',
       'builtin.prior-backup-check'
     )),
    '[]'::jsonb
  )
)
WHERE payload->>'sqlReviewRules' IS NOT NULL
  AND jsonb_array_length(payload->'sqlReviewRules') > 0;
