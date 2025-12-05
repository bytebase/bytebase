-- Migrate PASSWORD_RESTRICTION into WORKSPACE_PROFILE as passwordRestriction
-- Extract password restriction settings from PASSWORD_RESTRICTION and merge into WORKSPACE_PROFILE
UPDATE setting
SET value = jsonb_set(
    value,
    '{passwordRestriction}',
    (SELECT value FROM setting WHERE name = 'PASSWORD_RESTRICTION'),
    true
)
WHERE name = 'WORKSPACE_PROFILE'
AND EXISTS (
    SELECT 1 FROM setting WHERE name = 'PASSWORD_RESTRICTION'
);

-- Delete the old PASSWORD_RESTRICTION setting
DELETE FROM setting WHERE name = 'PASSWORD_RESTRICTION';
