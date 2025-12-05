-- Migrate BRANDING_LOGO into WORKSPACE_PROFILE as brandingLogo
-- Extract logo from BRANDING_LOGO setting and merge into WORKSPACE_PROFILE
UPDATE setting
SET value = jsonb_set(
    COALESCE(value::jsonb, '{}'::jsonb),
    '{brandingLogo}',
    to_jsonb((SELECT value FROM setting WHERE name = 'BRANDING_LOGO')),
    true
)::text
WHERE name = 'WORKSPACE_PROFILE'
AND EXISTS (
    SELECT 1 FROM setting WHERE name = 'BRANDING_LOGO' AND value IS NOT NULL AND value != ''
);

-- Delete the old BRANDING_LOGO setting
DELETE FROM setting WHERE name = 'BRANDING_LOGO';
