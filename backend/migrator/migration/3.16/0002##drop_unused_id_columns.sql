-- Drop unused id columns from tables that already use natural keys.
-- No other tables have FK references to these id columns.
-- For tables whose unique indexes are referenced by FK constraints from other tables
-- (project, instance, db), we promote the existing unique index to PK using
-- ALTER TABLE ... ADD CONSTRAINT ... PRIMARY KEY USING INDEX to avoid FK dependency errors.

DO $$
BEGIN
    -- project: natural key = resource_id (FK referenced by many tables)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'project' AND column_name = 'id') THEN
        ALTER TABLE project DROP CONSTRAINT project_pkey;
        ALTER TABLE project DROP COLUMN id;
        ALTER TABLE project ADD CONSTRAINT project_pkey PRIMARY KEY USING INDEX idx_project_unique_resource_id;
    END IF;

    -- instance: natural key = resource_id (FK referenced by db, task)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'instance' AND column_name = 'id') THEN
        ALTER TABLE instance DROP CONSTRAINT instance_pkey;
        ALTER TABLE instance DROP COLUMN id;
        ALTER TABLE instance ADD CONSTRAINT instance_pkey PRIMARY KEY USING INDEX idx_instance_unique_resource_id;
    END IF;

    -- db: natural key = (instance, name) (FK referenced by db_schema, revision, sync_history, changelog)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'db' AND column_name = 'id') THEN
        ALTER TABLE db DROP CONSTRAINT db_pkey;
        ALTER TABLE db DROP COLUMN id;
        ALTER TABLE db ADD CONSTRAINT db_pkey PRIMARY KEY USING INDEX idx_db_unique_instance_name;
    END IF;

    -- setting: natural key = name
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'setting' AND column_name = 'id') THEN
        ALTER TABLE setting DROP COLUMN id;
        ALTER TABLE setting ADD PRIMARY KEY (name);
        DROP INDEX IF EXISTS idx_setting_unique_name;
    END IF;

    -- policy: natural key = (resource_type, resource, type)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'policy' AND column_name = 'id') THEN
        ALTER TABLE policy DROP COLUMN id;
        ALTER TABLE policy ADD PRIMARY KEY (resource_type, resource, type);
        DROP INDEX IF EXISTS idx_policy_unique_resource_type_resource_type;
    END IF;

    -- idp: natural key = resource_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'idp' AND column_name = 'id') THEN
        ALTER TABLE idp DROP COLUMN id;
        ALTER TABLE idp ADD PRIMARY KEY (resource_id);
        DROP INDEX IF EXISTS idx_idp_unique_resource_id;
    END IF;

    -- role: natural key = resource_id
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'role' AND column_name = 'id') THEN
        ALTER TABLE role DROP COLUMN id;
        ALTER TABLE role ADD PRIMARY KEY (resource_id);
        DROP INDEX IF EXISTS idx_role_unique_resource_id;
    END IF;

    -- db_schema: natural key = (instance, db_name)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'db_schema' AND column_name = 'id') THEN
        ALTER TABLE db_schema DROP COLUMN id;
        ALTER TABLE db_schema ADD PRIMARY KEY (instance, db_name);
        DROP INDEX IF EXISTS idx_db_schema_unique_instance_db_name;
    END IF;

    -- db_group: natural key = (project, resource_id)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'db_group' AND column_name = 'id') THEN
        ALTER TABLE db_group DROP COLUMN id;
        ALTER TABLE db_group ADD PRIMARY KEY (project, resource_id);
        DROP INDEX IF EXISTS idx_db_group_unique_project_resource_id;
    END IF;

    -- task_run_log: use (task_run_id, created_at) as PK
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'task_run_log' AND column_name = 'id') THEN
        ALTER TABLE task_run_log DROP COLUMN id;
        ALTER TABLE task_run_log ADD PRIMARY KEY (task_run_id, created_at);
    END IF;

    -- release: natural key = (project, train, iteration)
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'release' AND column_name = 'id') THEN
        ALTER TABLE release DROP COLUMN id;
        ALTER TABLE release ADD PRIMARY KEY (project, train, iteration);
        DROP INDEX IF EXISTS idx_release_project_train_iteration;
    END IF;
END $$;
