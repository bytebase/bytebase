-- Change policy PK from (resource_type, resource, type) to
-- (workspace, resource_type, resource, type) for multi-workspace support.
-- The old PK rejected cross-workspace policies sharing the same
-- (resource_type, resource, type) triple — e.g. two workspaces each with a
-- TAG policy on `environments/prod` collided on policy_pkey on INSERT before
-- the workspace-scoped ON CONFLICT clause could route the upsert to UPDATE.
ALTER TABLE policy DROP CONSTRAINT IF EXISTS policy_pkey;
DO $$
BEGIN
    ALTER TABLE policy ADD PRIMARY KEY (workspace, resource_type, resource, type);
EXCEPTION WHEN duplicate_table THEN
    -- PK already exists.
END $$;
DROP INDEX IF EXISTS idx_policy_unique_workspace_resource;
