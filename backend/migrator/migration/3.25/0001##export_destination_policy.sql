-- Exports go only to destinations the workspace query data policy allows.
-- Every workspace keeps download, the only destination before this version.
UPDATE policy
SET payload = jsonb_set(payload, '{exportDestinationPolicy}', '{"allowedDestinations":["DOWNLOAD"]}')
WHERE type = 'QUERY_DATA' AND resource_type = 'WORKSPACE'
  AND NOT payload ? 'exportDestinationPolicy';

INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent, enforce)
SELECT resource_id, 'WORKSPACE', 'workspaces/' || resource_id, 'QUERY_DATA',
       '{"exportDestinationPolicy":{"allowedDestinations":["DOWNLOAD"]}}', FALSE, TRUE
FROM workspace
ON CONFLICT (workspace, resource_type, resource, type) DO NOTHING;
