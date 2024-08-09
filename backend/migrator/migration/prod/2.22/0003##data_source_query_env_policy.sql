INSERT INTO policy (creator_id, updater_id, type, payload, resource_type, resource_id, inherit_from_parent)
SELECT 
  1, 
  1, 
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
