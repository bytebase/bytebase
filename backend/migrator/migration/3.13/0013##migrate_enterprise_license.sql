-- Migrate ENTERPRISE_LICENSE to SYSTEM setting
-- Only backfill if the license string is not empty
UPDATE setting
SET value = jsonb_set(
  value::jsonb,
  '{license}',
  to_jsonb((SELECT value FROM setting WHERE name = 'ENTERPRISE_LICENSE'))
)::TEXT
WHERE name = 'SYSTEM'
  AND (SELECT value FROM setting WHERE name = 'ENTERPRISE_LICENSE') != '';

DELETE FROM setting WHERE name = 'ENTERPRISE_LICENSE';
