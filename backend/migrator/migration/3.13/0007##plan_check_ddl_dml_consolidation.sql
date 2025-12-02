-- Consolidate ChangeDatabaseType in plan_check_run configs.
-- DDL(1), DML(2), DDL_GHOST(4) -> CHANGE_DATABASE
-- SDL(3) -> SDL

-- Update DDL, DML, DDL_GHOST to CHANGE_DATABASE
UPDATE plan_check_run
SET config = jsonb_set(config, '{changeDatabaseType}', '"CHANGE_DATABASE"')
WHERE config->>'changeDatabaseType' IN ('DDL', 'DML', 'DDL_GHOST');

-- Update SDL to new value (stays as SDL but enum value changes from 3 to 2)
-- Note: protojson serializes as string, so we update the string value
UPDATE plan_check_run
SET config = jsonb_set(config, '{changeDatabaseType}', '"SDL"')
WHERE config->>'changeDatabaseType' = 'SDL';

-- Remove disallow-mix rules from review_config if they exist
UPDATE review_config
SET payload = jsonb_set(
  payload,
  '{rules}',
  COALESCE(
    (SELECT jsonb_agg(r)
     FROM jsonb_array_elements(payload->'rules') r
     WHERE r->>'type' NOT IN (
       'statement.disallow-mix-in-ddl',
       'statement.disallow-mix-in-dml'
     )),
    '[]'::jsonb
  )
)
WHERE payload->'rules' IS NOT NULL
  AND jsonb_array_length(payload->'rules') > 0;
