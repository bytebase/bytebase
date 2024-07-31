INSERT INTO policy (row_status, creator_id, created_ts, updater_id, updated_ts, type, payload, resource_type, resource_id, inherit_from_parent)
SELECT 
  'NORMAL', 
  101, 
  extract(epoch from now()), 
  101, 
  extract(epoch from now()), 
  'bb.policy.data-source-query', 
  '{"adminDataSourceRestriction": "FALLBACK"}', 
  'ENVIRONMENT', 
  e.id, 
  false
FROM 
    environment e
WHERE 
  NOT EXISTS (
    SELECT 1 
    FROM policy p 
    WHERE 
      p.resource_type = 'ENVIRONMENT' 
      AND p.resource_id = e.id 
      AND p.type = 'bb.policy.data-source-query'
  );
