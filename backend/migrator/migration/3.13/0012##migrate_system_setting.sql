-- Migrate AUTH_SECRET and WORKSPACE_ID to SYSTEM setting
INSERT INTO setting (name, value)
SELECT
  'SYSTEM',
  json_build_object(
    'authSecret', (SELECT value FROM setting WHERE name = 'AUTH_SECRET'),
    'workspaceId', (SELECT value FROM setting WHERE name = 'WORKSPACE_ID')
  )::TEXT;

DELETE FROM setting WHERE name IN ('AUTH_SECRET', 'WORKSPACE_ID');
