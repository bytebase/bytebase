-- Migrate AlertLevel enum values after removing ALERT_LEVEL_ prefix
-- Old values: ALERT_LEVEL_INFO, ALERT_LEVEL_WARNING, ALERT_LEVEL_CRITICAL
-- New values: INFO, WARNING, CRITICAL

UPDATE setting
SET value = jsonb_set(
    value,
    '{announcement,level}',
    CASE
        WHEN value -> 'announcement' ->> 'level' = 'ALERT_LEVEL_INFO' THEN '"INFO"'::jsonb
        WHEN value -> 'announcement' ->> 'level' = 'ALERT_LEVEL_WARNING' THEN '"WARNING"'::jsonb
        WHEN value -> 'announcement' ->> 'level' = 'ALERT_LEVEL_CRITICAL' THEN '"CRITICAL"'::jsonb
        ELSE value -> 'announcement' -> 'level'
    END
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value -> 'announcement' ->> 'level' IN ('ALERT_LEVEL_INFO', 'ALERT_LEVEL_WARNING', 'ALERT_LEVEL_CRITICAL');
