-- Migrate WATERMARK setting into WORKSPACE_PROFILE
-- Only set watermark to true if the value is exactly "1", otherwise rely on proto3 default (false)
UPDATE setting
SET value = jsonb_set(
    COALESCE(value::jsonb, '{}'::jsonb),
    '{watermark}',
    'true'::jsonb,
    true
)::text
WHERE name = 'WORKSPACE_PROFILE'
AND EXISTS (
    SELECT 1 FROM setting WHERE name = 'WATERMARK' AND value = '1'
);

-- Delete the old WATERMARK setting
DELETE FROM setting WHERE name = 'WATERMARK';
