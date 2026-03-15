-- Drop unused id columns from tables that already use natural keys.
-- No other tables have FK references to these id columns.

-- project: natural key = resource_id (FK referenced by many tables)
ALTER TABLE project DROP CONSTRAINT IF EXISTS project_pkey;
ALTER TABLE project DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_project_unique_resource_id;
ALTER TABLE project ADD PRIMARY KEY (resource_id);

-- instance: natural key = resource_id (FK referenced by db, task)
ALTER TABLE instance DROP CONSTRAINT IF EXISTS instance_pkey;
ALTER TABLE instance DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_instance_unique_resource_id;
ALTER TABLE instance ADD PRIMARY KEY (resource_id);

-- db: natural key = (instance, name) (FK referenced by db_schema, revision, sync_history, changelog)
ALTER TABLE db DROP CONSTRAINT IF EXISTS db_pkey;
ALTER TABLE db DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_db_unique_instance_name;
ALTER TABLE db ADD PRIMARY KEY (instance, name);

-- setting: natural key = name
ALTER TABLE setting DROP CONSTRAINT IF EXISTS setting_pkey;
ALTER TABLE setting DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_setting_unique_name;
ALTER TABLE setting ADD PRIMARY KEY (name);

-- policy: natural key = (resource_type, resource, type)
ALTER TABLE policy DROP CONSTRAINT IF EXISTS policy_pkey;
ALTER TABLE policy DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_policy_unique_resource_type_resource_type;
ALTER TABLE policy ADD PRIMARY KEY (resource_type, resource, type);

-- idp: natural key = resource_id
ALTER TABLE idp DROP CONSTRAINT IF EXISTS idp_pkey;
ALTER TABLE idp DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_idp_unique_resource_id;
ALTER TABLE idp ADD PRIMARY KEY (resource_id);

-- role: natural key = resource_id
ALTER TABLE role DROP CONSTRAINT IF EXISTS role_pkey;
ALTER TABLE role DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_role_unique_resource_id;
ALTER TABLE role ADD PRIMARY KEY (resource_id);

-- db_schema: natural key = (instance, db_name)
ALTER TABLE db_schema DROP CONSTRAINT IF EXISTS db_schema_pkey;
ALTER TABLE db_schema DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_db_schema_unique_instance_db_name;
ALTER TABLE db_schema ADD PRIMARY KEY (instance, db_name);

-- db_group: natural key = (project, resource_id)
ALTER TABLE db_group DROP CONSTRAINT IF EXISTS db_group_pkey;
ALTER TABLE db_group DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_db_group_unique_project_resource_id;
ALTER TABLE db_group ADD PRIMARY KEY (project, resource_id);

-- task_run_log: use (task_run_id, created_at) as PK
ALTER TABLE task_run_log DROP CONSTRAINT IF EXISTS task_run_log_pkey;
ALTER TABLE task_run_log DROP COLUMN IF EXISTS id;
ALTER TABLE task_run_log ADD PRIMARY KEY (task_run_id, created_at);

-- release: natural key = (project, train, iteration)
ALTER TABLE release DROP CONSTRAINT IF EXISTS release_pkey;
ALTER TABLE release DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_release_project_train_iteration;
ALTER TABLE release ADD PRIMARY KEY (project, train, iteration);
