-- Drop unused id columns from tables that already use natural keys.
-- No other tables have FK references to these id columns.

-- project: natural key = resource_id (FK referenced by many tables)
ALTER TABLE project DROP CONSTRAINT IF EXISTS project_pkey;
ALTER TABLE project DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_project_unique_resource_id') IS NOT NULL THEN
        ALTER TABLE project ADD CONSTRAINT project_pkey PRIMARY KEY USING INDEX idx_project_unique_resource_id;
    ELSE
        ALTER TABLE project ADD PRIMARY KEY (resource_id);
    END IF;
END $$;

-- instance: natural key = resource_id (FK referenced by db, task)
ALTER TABLE instance DROP CONSTRAINT IF EXISTS instance_pkey;
ALTER TABLE instance DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_instance_unique_resource_id') IS NOT NULL THEN
        ALTER TABLE instance ADD CONSTRAINT instance_pkey PRIMARY KEY USING INDEX idx_instance_unique_resource_id;
    ELSE
        ALTER TABLE instance ADD PRIMARY KEY (resource_id);
    END IF;
END $$;

-- db: natural key = (instance, name) (FK referenced by db_schema, revision, sync_history, changelog)
ALTER TABLE db DROP CONSTRAINT IF EXISTS db_pkey;
ALTER TABLE db DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_db_unique_instance_name') IS NOT NULL THEN
        ALTER TABLE db ADD CONSTRAINT db_pkey PRIMARY KEY USING INDEX idx_db_unique_instance_name;
    ELSE
        ALTER TABLE db ADD PRIMARY KEY (instance, name);
    END IF;
END $$;

-- setting: natural key = name
ALTER TABLE setting DROP CONSTRAINT IF EXISTS setting_pkey;
ALTER TABLE setting DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_setting_unique_name') IS NOT NULL THEN
        ALTER TABLE setting ADD CONSTRAINT setting_pkey PRIMARY KEY USING INDEX idx_setting_unique_name;
    ELSE
        ALTER TABLE setting ADD PRIMARY KEY (name);
    END IF;
END $$;

-- policy: natural key = (resource_type, resource, type)
ALTER TABLE policy DROP CONSTRAINT IF EXISTS policy_pkey;
ALTER TABLE policy DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_policy_unique_resource_type_resource_type') IS NOT NULL THEN
        ALTER TABLE policy ADD CONSTRAINT policy_pkey PRIMARY KEY USING INDEX idx_policy_unique_resource_type_resource_type;
    ELSE
        ALTER TABLE policy ADD PRIMARY KEY (resource_type, resource, type);
    END IF;
END $$;

-- idp: natural key = resource_id
ALTER TABLE idp DROP CONSTRAINT IF EXISTS idp_pkey;
ALTER TABLE idp DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_idp_unique_resource_id') IS NOT NULL THEN
        ALTER TABLE idp ADD CONSTRAINT idp_pkey PRIMARY KEY USING INDEX idx_idp_unique_resource_id;
    ELSE
        ALTER TABLE idp ADD PRIMARY KEY (resource_id);
    END IF;
END $$;

-- role: natural key = resource_id
ALTER TABLE role DROP CONSTRAINT IF EXISTS role_pkey;
ALTER TABLE role DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_role_unique_resource_id') IS NOT NULL THEN
        ALTER TABLE role ADD CONSTRAINT role_pkey PRIMARY KEY USING INDEX idx_role_unique_resource_id;
    ELSE
        ALTER TABLE role ADD PRIMARY KEY (resource_id);
    END IF;
END $$;

-- db_schema: natural key = (instance, db_name)
ALTER TABLE db_schema DROP CONSTRAINT IF EXISTS db_schema_pkey;
ALTER TABLE db_schema DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_db_schema_unique_instance_db_name') IS NOT NULL THEN
        ALTER TABLE db_schema ADD CONSTRAINT db_schema_pkey PRIMARY KEY USING INDEX idx_db_schema_unique_instance_db_name;
    ELSE
        ALTER TABLE db_schema ADD PRIMARY KEY (instance, db_name);
    END IF;
END $$;

-- db_group: natural key = (project, resource_id)
ALTER TABLE db_group DROP CONSTRAINT IF EXISTS db_group_pkey;
ALTER TABLE db_group DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_db_group_unique_project_resource_id') IS NOT NULL THEN
        ALTER TABLE db_group ADD CONSTRAINT db_group_pkey PRIMARY KEY USING INDEX idx_db_group_unique_project_resource_id;
    ELSE
        ALTER TABLE db_group ADD PRIMARY KEY (project, resource_id);
    END IF;
END $$;

-- task_run_log: use (task_run_id, created_at) as PK
ALTER TABLE task_run_log DROP CONSTRAINT IF EXISTS task_run_log_pkey;
ALTER TABLE task_run_log DROP COLUMN IF EXISTS id;
ALTER TABLE task_run_log ADD PRIMARY KEY (task_run_id, created_at);

-- release: natural key = (project, train, iteration)
ALTER TABLE release DROP CONSTRAINT IF EXISTS release_pkey;
ALTER TABLE release DROP COLUMN IF EXISTS id;
DO $$
BEGIN
    IF to_regclass('idx_release_project_train_iteration') IS NOT NULL THEN
        ALTER TABLE release ADD CONSTRAINT release_pkey PRIMARY KEY USING INDEX idx_release_project_train_iteration;
    ELSE
        ALTER TABLE release ADD PRIMARY KEY (project, train, iteration);
    END IF;
END $$;
