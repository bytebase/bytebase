INSERT INTO policy (
  type,
  resource_type,
  resource,
  inherit_from_parent,
  payload
) 
VALUES (
  'bb.policy.query-data',
  'WORKSPACE',
  '',
  false,
  '{"timeout": "600s"}'::jsonb
)
ON CONFLICT (resource_type, resource, type) DO NOTHING;