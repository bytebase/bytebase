-- Rename the workspace profile `maximumRoleExpiration` JSON key to
-- `maximumRequestExpiration`, following the proto field rename. The cap now
-- bounds both role grants and data access requests, so the broader name fits.
--
-- Safety:
--   * Only touches a WORKSPACE_PROFILE row that actually carries the old key,
--     so it never fabricates a cap on workspaces that never set one.
--   * Idempotent: the run removes the old key, and the WHERE excludes rows that
--     no longer have it, so a re-run matches nothing.
UPDATE setting
SET value = jsonb_set(
    value #- '{maximumRoleExpiration}',
    '{maximumRequestExpiration}',
    value->'maximumRoleExpiration',
    true
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'maximumRoleExpiration';
