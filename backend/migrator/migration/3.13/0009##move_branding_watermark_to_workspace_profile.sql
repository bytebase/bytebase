-- Move BRANDING_LOGO and WATERMARK settings into WORKSPACE_PROFILE setting
UPDATE setting
SET value = (
    COALESCE(value::jsonb, '{}'::jsonb)
    || CASE
        WHEN EXISTS (SELECT 1 FROM setting WHERE name = 'BRANDING_LOGO' AND value != '')
        THEN jsonb_build_object('brandingLogo', (SELECT value FROM setting WHERE name = 'BRANDING_LOGO'))
        ELSE '{}'::jsonb
       END
    || CASE
        WHEN EXISTS (SELECT 1 FROM setting WHERE name = 'WATERMARK' AND value = '1')
        THEN jsonb_build_object('watermark', true)
        ELSE '{}'::jsonb
       END
)::text
WHERE name = 'WORKSPACE_PROFILE';

DELETE FROM setting WHERE name IN ('BRANDING_LOGO', 'WATERMARK');
