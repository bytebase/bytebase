-- Rename tokenDuration to refreshTokenDuration in WORKSPACE_PROFILE setting
UPDATE setting
SET value = jsonb_set(
  value - 'tokenDuration',
  '{refreshTokenDuration}',
  value->'tokenDuration'
)
WHERE name = 'WORKSPACE_PROFILE'
  AND value ? 'tokenDuration';
