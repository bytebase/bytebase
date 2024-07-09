INSERT INTO db_group (creator_id, updater_id, project_id, resource_id, placeholder, expression, payload)
SELECT 
    p.creator_id,
    p.updater_id,
    p.id,
    'all-databases' AS resource_id,
    'all-databases' AS placeholder,
    '{"expression": "true"}'::jsonb AS expression,
    '{"multitenancy": true}'::jsonb AS payload
FROM 
    project p
WHERE 
    p.tenant_mode = 'TENANT';
