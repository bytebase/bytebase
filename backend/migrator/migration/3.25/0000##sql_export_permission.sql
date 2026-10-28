DO $$
DECLARE
  conflicting_workspace text;
BEGIN
  SELECT workspace INTO conflicting_workspace FROM role WHERE resource_id = 'sqlEditorExporter';
  IF FOUND THEN
    RAISE EXCEPTION 'custom role "sqlEditorExporter" in workspace "%" conflicts with a new built-in role. On the previous Bytebase version, create a replacement custom role, move its bindings, and delete "sqlEditorExporter".', conflicting_workspace;
  END IF;
END $$;

CREATE TEMP TABLE approval_only_workspace ON COMMIT DROP AS
SELECT workspace FROM policy
WHERE resource_type = 'WORKSPACE' AND type = 'QUERY_DATA'
  AND COALESCE((payload ->> 'disableExport')::boolean, false);

WITH added AS (
  SELECT p.workspace, p.resource_type, p.resource,
         jsonb_agg(jsonb_set(b, '{role}', '"roles/sqlEditorExporter"') ORDER BY ord) AS bindings
  FROM policy p,
       jsonb_array_elements(p.payload -> 'bindings') WITH ORDINALITY AS t(b, ord)
  WHERE p.type = 'IAM'
    AND p.resource_type IN ('WORKSPACE', 'PROJECT')
    AND p.workspace NOT IN (SELECT workspace FROM approval_only_workspace)
    AND b ->> 'role' IN ('roles/sqlEditorUser', 'roles/sqlEditorReadUser')
  GROUP BY p.workspace, p.resource_type, p.resource
)
UPDATE policy p
SET payload = jsonb_set(p.payload, '{bindings}', (p.payload -> 'bindings') || added.bindings)
FROM added
WHERE p.type = 'IAM' AND p.workspace = added.workspace
  AND p.resource_type = added.resource_type AND p.resource = added.resource;

UPDATE policy
SET payload = payload - 'disableExport'
WHERE type = 'QUERY_DATA' AND payload ? 'disableExport';
