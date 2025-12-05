-- Migrate SCIM token into WORKSPACE_PROFILE as directorySyncToken
-- Extract token from SCIM setting and merge into WORKSPACE_PROFILE
UPDATE setting
SET value = jsonb_set(
    COALESCE(value::jsonb, '{}'::jsonb),
    '{directorySyncToken}',
    to_jsonb((SELECT value::jsonb->>'token' FROM setting WHERE name = 'SCIM')),
    true
)::text
WHERE name = 'WORKSPACE_PROFILE'
AND EXISTS (
    SELECT 1 FROM setting WHERE name = 'SCIM' AND value::jsonb->>'token' IS NOT NULL AND value::jsonb->>'token' != ''
);

-- Delete the old SCIM setting
DELETE FROM setting WHERE name = 'SCIM';
