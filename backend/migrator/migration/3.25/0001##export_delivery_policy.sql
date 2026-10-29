-- Exports go only to deliveries the workspace query data policy allows. Every
-- workspace keeps local download, the only delivery before this version.
UPDATE policy
SET payload = jsonb_set(payload, '{exportDeliveryPolicy}', '{"allowedDeliveries":["LOCAL"]}')
WHERE type = 'QUERY_DATA' AND resource_type = 'WORKSPACE'
  AND NOT payload ? 'exportDeliveryPolicy';

INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent, enforce)
SELECT resource_id, 'WORKSPACE', 'workspaces/' || resource_id, 'QUERY_DATA',
       '{"exportDeliveryPolicy":{"allowedDeliveries":["LOCAL"]}}', FALSE, TRUE
FROM workspace
ON CONFLICT (workspace, resource_type, resource, type) DO NOTHING;
