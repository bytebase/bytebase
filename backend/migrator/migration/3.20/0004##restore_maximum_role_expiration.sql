-- Restore the workspace profile `maximumRoleExpiration` JSON key while keeping
-- `maximumRequestExpiration` for request access.
--
-- Safety:
--   * Only copies from an existing `maximumRequestExpiration` key, so it never
--     fabricates a cap for workspaces without a configured request expiration.
--   * Does not overwrite a `maximumRoleExpiration` key if it already exists.
--   * Idempotent: once the restored key exists, the WHERE clause excludes it.
UPDATE setting
SET value = jsonb_set(
    value,
    '{maximumRoleExpiration}',
    value->'maximumRequestExpiration',
    true
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'maximumRequestExpiration'
  AND NOT (value ? 'maximumRoleExpiration');
