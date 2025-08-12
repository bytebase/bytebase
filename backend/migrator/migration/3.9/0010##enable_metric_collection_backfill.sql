-- Backfill enable_metric_collection to true for existing workspace profile settings
UPDATE setting
SET value = jsonb_set(
    value::jsonb,
    '{enableMetricCollection}',
    'true'::jsonb,
    true
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value::jsonb IS NOT NULL
  AND NOT (value::jsonb ? 'enableMetricCollection');