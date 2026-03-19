-- Step 02: Create server_config, workspace table, add workspace to root tables, update indexes.

------------------------------
-- 0. Create server_config and migrate auth_secret from setting
------------------------------
CREATE TABLE IF NOT EXISTS server_config (
    -- Stored as ServerConfigPayload (proto/store/store/server_config.proto)
    payload     jsonb NOT NULL DEFAULT '{}'
);

-- Migrate auth_secret from SYSTEM setting to server_config (only if empty).
INSERT INTO server_config (payload)
SELECT json_build_object('authSecret', COALESCE(value->>'authSecret', ''))
FROM setting
WHERE name = 'SYSTEM'
  AND NOT EXISTS (SELECT 1 FROM server_config)
LIMIT 1;

------------------------------
-- 1. Create workspace table
------------------------------
CREATE TABLE IF NOT EXISTS workspace (
    resource_id text PRIMARY KEY,
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    deleted     boolean NOT NULL DEFAULT FALSE
);

-- Seed the default workspace from the existing SYSTEM setting (only if no workspace exists).
INSERT INTO workspace (resource_id, name)
SELECT
    COALESCE(NULLIF(value->>'workspaceId', ''), gen_random_uuid()::text),
    'Default Workspace'
FROM setting
WHERE name = 'SYSTEM'
  AND NOT EXISTS (SELECT 1 FROM workspace)
LIMIT 1;

----------------------------------------------
-- 2. Add workspace to root tables
----------------------------------------------
-- Add workspace column to root tables using NOT NULL DEFAULT to avoid table rewrite.
-- PostgreSQL 11+ stores the default in pg_attribute — instant for large tables like audit_log.
DO $$
DECLARE
    t text;
    ws_id text;
BEGIN
    SELECT resource_id INTO ws_id FROM workspace LIMIT 1;
    IF ws_id IS NULL THEN
        RAISE EXCEPTION 'no workspace found';
    END IF;

    FOREACH t IN ARRAY ARRAY[
        'project', 'instance', 'setting', 'policy', 'role', 'idp',
        'review_config', 'user_group', 'export_archive', 'audit_log',
        'service_account', 'workload_identity', 'oauth2_client'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = t AND column_name = 'workspace') THEN
            -- ADD COLUMN ... NOT NULL DEFAULT is instant in PG11+ (no table rewrite).
            EXECUTE format('ALTER TABLE %I ADD COLUMN workspace text NOT NULL DEFAULT %L', t, ws_id);
            -- Add FK constraint separately (doesn't require table rewrite).
            EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I FOREIGN KEY (workspace) REFERENCES workspace(resource_id)', t, t || '_workspace_fkey');
            -- Remove the default so future inserts must provide workspace explicitly.
            EXECUTE format('ALTER TABLE %I ALTER COLUMN workspace DROP DEFAULT', t);
        END IF;
    END LOOP;
END $$;

----------------------------------------------
-- 3. Update unique indexes
----------------------------------------------
-- PKs (resource_id) are already globally unique — no composite unique index needed for
-- project, instance, role, idp.
-- Only tables where the PK is NOT the resource_id need workspace-scoped unique indexes.

-- setting: PK is name, need workspace-scoped uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_setting_unique_workspace_name ON setting(workspace, name);

-- user_group: replace existing email unique index with workspace-scoped one
DROP INDEX IF EXISTS idx_user_group_unique_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_group_unique_email ON user_group(workspace, email) WHERE email IS NOT NULL;

-- service_account
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_account_unique_workspace_email ON service_account(workspace, email);

-- workload_identity
CREATE UNIQUE INDEX IF NOT EXISTS idx_workload_identity_unique_workspace_email ON workload_identity(workspace, email);

----------------------------------------------
-- 4. Workspace query performance indexes
----------------------------------------------
CREATE INDEX IF NOT EXISTS idx_project_workspace ON project(workspace);
CREATE INDEX IF NOT EXISTS idx_instance_workspace ON instance(workspace);
CREATE INDEX IF NOT EXISTS idx_setting_workspace ON setting(workspace);
CREATE INDEX IF NOT EXISTS idx_policy_workspace ON policy(workspace);
CREATE UNIQUE INDEX IF NOT EXISTS idx_policy_unique_workspace_resource ON policy(workspace, resource_type, resource, type);
CREATE INDEX IF NOT EXISTS idx_audit_log_workspace ON audit_log(workspace);
CREATE INDEX IF NOT EXISTS idx_service_account_workspace ON service_account(workspace);
CREATE INDEX IF NOT EXISTS idx_workload_identity_workspace ON workload_identity(workspace);
CREATE INDEX IF NOT EXISTS idx_oauth2_client_workspace ON oauth2_client(workspace);
