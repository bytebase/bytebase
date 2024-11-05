INSERT INTO policy (
  creator_id,
  updater_id,
  type,
  resource_type,
  resource_id,
  inherit_from_parent,
  payload
) SELECT 
  1,
  1,
  'bb.policy.data-source-query',
  'ENVIRONMENT',
  environment.id,
  false,
  '{"disableDdl": true, "disableDml": true, "adminDataSourceRestriction": "FALLBACK"}'
FROM environment
ON CONFLICT (resource_type, resource_id, type)
DO UPDATE SET payload = policy.payload || '{"disableDdl": true, "disableDml": true}'