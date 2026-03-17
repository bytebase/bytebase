-- Step 02: Create workspace table, add workspace to root tables, update indexes.

------------------------------
-- 1. Create workspace table
------------------------------
CREATE TABLE workspace (
    resource_id text PRIMARY KEY,
    name        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    deleted     boolean NOT NULL DEFAULT FALSE
);

-- Seed the default workspace from the existing SYSTEM setting.
INSERT INTO workspace (resource_id, name)
SELECT
    COALESCE(NULLIF(value->>'workspaceId', ''), gen_random_uuid()::text),
    'Default Workspace'
FROM setting
WHERE name = 'SYSTEM';

----------------------------------------------
-- 2. Add workspace to 12 root tables
----------------------------------------------

-- project
ALTER TABLE project ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE project SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE project ALTER COLUMN workspace SET NOT NULL;

-- instance
ALTER TABLE instance ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE instance SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE instance ALTER COLUMN workspace SET NOT NULL;

-- setting
ALTER TABLE setting ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE setting SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE setting ALTER COLUMN workspace SET NOT NULL;

-- policy
ALTER TABLE policy ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE policy SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE policy ALTER COLUMN workspace SET NOT NULL;

-- role
ALTER TABLE role ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE role SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE role ALTER COLUMN workspace SET NOT NULL;

-- idp
ALTER TABLE idp ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE idp SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE idp ALTER COLUMN workspace SET NOT NULL;

-- review_config
ALTER TABLE review_config ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE review_config SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE review_config ALTER COLUMN workspace SET NOT NULL;

-- user_group
ALTER TABLE user_group ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE user_group SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE user_group ALTER COLUMN workspace SET NOT NULL;

-- export_archive
ALTER TABLE export_archive ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE export_archive SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE export_archive ALTER COLUMN workspace SET NOT NULL;

-- audit_log
ALTER TABLE audit_log ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE audit_log SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE audit_log ALTER COLUMN workspace SET NOT NULL;

-- service_account
ALTER TABLE service_account ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE service_account SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE service_account ALTER COLUMN workspace SET NOT NULL;

-- workload_identity
ALTER TABLE workload_identity ADD COLUMN workspace text REFERENCES workspace(resource_id);
UPDATE workload_identity SET workspace = (SELECT resource_id FROM workspace LIMIT 1);
ALTER TABLE workload_identity ALTER COLUMN workspace SET NOT NULL;

----------------------------------------------
-- 3. Update unique indexes
----------------------------------------------
-- Current PKs (resource_id, name, email) already enforce global uniqueness.
-- These composite unique indexes prepare for multi-workspace where uniqueness
-- will be workspace-scoped (after a future PK migration).

-- project
CREATE UNIQUE INDEX idx_project_unique_workspace_resource_id ON project(workspace, resource_id);

-- instance
CREATE UNIQUE INDEX idx_instance_unique_workspace_resource_id ON instance(workspace, resource_id);

-- setting
CREATE UNIQUE INDEX idx_setting_unique_workspace_name ON setting(workspace, name);

-- role
CREATE UNIQUE INDEX idx_role_unique_workspace_resource_id ON role(workspace, resource_id);

-- idp
CREATE UNIQUE INDEX idx_idp_unique_workspace_resource_id ON idp(workspace, resource_id);

-- user_group: replace existing email unique index with workspace-scoped one
DROP INDEX idx_user_group_unique_email;
CREATE UNIQUE INDEX idx_user_group_unique_email ON user_group(workspace, email) WHERE email IS NOT NULL;

-- service_account
CREATE UNIQUE INDEX idx_service_account_unique_workspace_email ON service_account(workspace, email);

-- workload_identity
CREATE UNIQUE INDEX idx_workload_identity_unique_workspace_email ON workload_identity(workspace, email);

----------------------------------------------
-- 4. Workspace query performance indexes
----------------------------------------------
CREATE INDEX idx_project_workspace ON project(workspace);
CREATE INDEX idx_instance_workspace ON instance(workspace);
CREATE INDEX idx_setting_workspace ON setting(workspace);
CREATE INDEX idx_policy_workspace ON policy(workspace);
CREATE INDEX idx_audit_log_workspace ON audit_log(workspace);
CREATE INDEX idx_service_account_workspace ON service_account(workspace);
CREATE INDEX idx_workload_identity_workspace ON workload_identity(workspace);
