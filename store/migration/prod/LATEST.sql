-- Type
CREATE TYPE row_status AS ENUM ('NORMAL', 'ARCHIVED');

-- updated_ts trigger.
CREATE OR REPLACE FUNCTION trigger_update_updated_ts()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_ts = extract(epoch from now());
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- principal
CREATE TABLE principal (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    type TEXT NOT NULL CHECK (type IN ('END_USER', 'SYSTEM_BOT')),
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_principal_unique_email ON principal(email);

CREATE TRIGGER update_principal_updated_ts
BEFORE
UPDATE
    ON principal FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Default bytebase system account id is 1
INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        type,
        name,
        email,
        password_hash
    )
VALUES
    (
        1,
        1,
        1,
        'SYSTEM_BOT',
        'Bytebase',
        'support@bytebase.com',
        ''
    );

ALTER SEQUENCE principal_id_seq RESTART WITH 101;

-- Setting
CREATE TABLE setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_setting_unique_name ON setting(name);

ALTER SEQUENCE setting_id_seq RESTART WITH 101;

CREATE TRIGGER update_setting_updated_ts
BEFORE
UPDATE
    ON setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Member
-- We separate the concept from Principal because if we support multiple workspace in the future, each workspace can have different member for the same principal
CREATE TABLE member (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    status TEXT NOT NULL CHECK (status IN ('INVITED', 'ACTIVE')),
    role TEXT NOT NULL CHECK (role IN ('OWNER', 'DBA', 'DEVELOPER')),
    principal_id INTEGER NOT NULL REFERENCES principal (id)
);

CREATE UNIQUE INDEX idx_member_unique_principal_id ON member(principal_id);

ALTER SEQUENCE member_id_seq RESTART WITH 101;

CREATE TRIGGER update_member_updated_ts
BEFORE
UPDATE
    ON member FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Environment
CREATE TABLE environment (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    "order" INTEGER NOT NULL CHECK ("order" >= 0)
);

CREATE UNIQUE INDEX idx_environment_unique_name ON environment(name);

ALTER SEQUENCE environment_id_seq RESTART WITH 101;

CREATE TRIGGER update_environment_updated_ts
BEFORE
UPDATE
    ON environment FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Policy
-- policy stores the policies for each environment.
-- Policies are associated with environments. Since we may have policies not associated with environment later, we name the table policy.
CREATE TABLE policy (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.policy.%'),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_policy_environment_id ON policy(environment_id);

CREATE UNIQUE INDEX idx_policy_unique_environment_id_type ON policy(environment_id, type);

ALTER SEQUENCE policy_id_seq RESTART WITH 101;

CREATE TRIGGER update_policy_updated_ts
BEFORE
UPDATE
    ON policy FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project
CREATE TABLE project (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    key TEXT NOT NULL,
    workflow_type TEXT NOT NULL CHECK (workflow_type IN ('UI', 'VCS')),
    visibility TEXT NOT NULL CHECK (visibility IN ('PUBLIC', 'PRIVATE')),
    tenant_mode TEXT NOT NULL CHECK (tenant_mode IN ('DISABLED', 'TENANT')) DEFAULT 'DISABLED',
    -- db_name_template is only used when a project is in tenant mode.
    -- Empty value means {{DB_NAME}}.
    db_name_template TEXT NOT NULL,
    role_provider TEXT NOT NULL CHECK (role_provider IN ('BYTEBASE', 'GITLAB_SELF_HOST')) DEFAULT 'BYTEBASE'
);

CREATE UNIQUE INDEX idx_project_unique_key ON project(key);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        key,
        workflow_type,
        visibility,
        tenant_mode,
        db_name_template
    )
VALUES
    (
        1,
        1,
        1,
        'Default',
        'DEFAULT',
        'UI',
        'PUBLIC',
        'DISABLED',
        ''
    );

ALTER SEQUENCE project_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_updated_ts
BEFORE
UPDATE
    ON project FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project member
CREATE TABLE project_member (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    role TEXT NOT NULL CHECK (role IN ('OWNER', 'DEVELOPER')),
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    role_provider TEXT NOT NULL CHECK (role_provider IN ('BYTEBASE', 'GITLAB_SELF_HOST')) DEFAULT 'BYTEBASE',
    -- payload is determined by the type of role_provider
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_project_member_unique_project_id_role_provider_principal_id ON project_member(project_id, role_provider, principal_id);

ALTER SEQUENCE project_member_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_member_updated_ts
BEFORE
UPDATE
    ON project_member FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Project Hook
CREATE TABLE project_webhook (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.plugin.webhook.%'),
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    activity_list TEXT ARRAY NOT NULL
);

CREATE INDEX idx_project_webhook_project_id ON project_webhook(project_id);

CREATE UNIQUE INDEX idx_project_webhook_unique_project_id_url ON project_webhook(project_id, url);

ALTER SEQUENCE project_webhook_id_seq RESTART WITH 101;

CREATE TRIGGER update_project_webhook_updated_ts
BEFORE
UPDATE
    ON project_webhook FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Instance
CREATE TABLE instance (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL,
    engine TEXT NOT NULL CHECK (engine IN ('MYSQL', 'POSTGRES', 'TIDB', 'CLICKHOUSE', 'SNOWFLAKE', 'SQLITE')),
    engine_version TEXT NOT NULL DEFAULT '',
    host TEXT NOT NULL,
    port TEXT NOT NULL,
    external_link TEXT NOT NULL DEFAULT ''
);

ALTER SEQUENCE instance_id_seq RESTART WITH 101;

CREATE TRIGGER update_instance_updated_ts
BEFORE
UPDATE
    ON instance FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Instance user stores the users for a particular instance
CREATE TABLE instance_user (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    name TEXT NOT NULL,
    "grant" TEXT NOT NULL
);

ALTER SEQUENCE instance_user_id_seq RESTART WITH 101;

CREATE UNIQUE INDEX idx_instance_user_unique_instance_id_name ON instance_user(instance_id, name);

CREATE TRIGGER update_instance_user_updated_ts
BEFORE
UPDATE
    ON instance_user FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db stores the databases for a particular instance
-- data is synced periodically from the instance
CREATE TABLE db (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    -- If db is restored from a backup, then we will record that backup id. We can thus trace up to the original db.
    source_backup_id INTEGER,
    sync_status TEXT NOT NULL CHECK (sync_status IN ('OK', 'NOT_FOUND')),
    last_successful_sync_ts BIGINT NOT NULL,
    schema_version TEXT NOT NULL,
    name TEXT NOT NULL,
    character_set TEXT NOT NULL,
    "collation" TEXT NOT NULL
);

CREATE INDEX idx_db_instance_id ON db(instance_id);

CREATE UNIQUE INDEX idx_db_unique_instance_id_name ON db(instance_id, name);

ALTER SEQUENCE db_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_updated_ts
BEFORE
UPDATE
    ON db FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- tbl stores the table for a particular database
-- data is synced periodically from the instance
CREATE TABLE tbl (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    engine TEXT NOT NULL,
    "collation" TEXT NOT NULL,
    row_count BIGINT NOT NULL,
    data_size BIGINT NOT NULL,
    index_size BIGINT NOT NULL,
    data_free BIGINT NOT NULL,
    create_options TEXT NOT NULL,
    comment TEXT NOT NULL
);

CREATE INDEX idx_tbl_database_id ON tbl(database_id);

CREATE UNIQUE INDEX idx_tbl_unique_database_id_name ON tbl(database_id, name);

ALTER SEQUENCE tbl_id_seq RESTART WITH 101;

CREATE TRIGGER update_tbl_updated_ts
BEFORE
UPDATE
    ON tbl FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- col stores the column for a particular table from a particular database
-- data is synced periodically from the instance
CREATE TABLE col (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    table_id INTEGER NOT NULL REFERENCES tbl (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    "default" TEXT,
    nullable BOOLEAN NOT NULL,
    type TEXT NOT NULL,
    character_set TEXT NOT NULL,
    "collation" TEXT NOT NULL,
    comment TEXT NOT NULL
);

CREATE INDEX idx_col_database_id_table_id ON col(database_id, table_id);

CREATE UNIQUE INDEX idx_col_unique_database_id_table_id_name ON col(database_id, table_id, name);

ALTER SEQUENCE col_id_seq RESTART WITH 101;

CREATE TRIGGER update_col_updated_ts
BEFORE
UPDATE
    ON col FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- idx stores the index for a particular table from a particular database
-- data is synced periodically from the instance
CREATE TABLE idx (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    table_id INTEGER NOT NULL REFERENCES tbl (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    expression TEXT NOT NULL,
    position INTEGER NOT NULL,
    type TEXT NOT NULL,
    "unique" BOOLEAN NOT NULL,
    visible BOOLEAN NOT NULL,
    comment TEXT NOT NULL
);

CREATE INDEX idx_idx_database_id_table_id ON idx(database_id, table_id);

CREATE UNIQUE INDEX idx_idx_unique_database_id_table_id_name_expression ON idx(database_id, table_id, name, expression);

ALTER SEQUENCE idx_id_seq RESTART WITH 101;

CREATE TRIGGER update_idx_updated_ts
BEFORE
UPDATE
    ON idx FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db_extension stores the extensions for a particular database.
-- data is synced periodically from the instance.
CREATE TABLE db_extension (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    schema TEXT NOT NULL,
    description TEXT NOT NULL
);

CREATE INDEX idx_db_extension_database_id ON db_extension(database_id);

CREATE UNIQUE INDEX idx_db_extension_unique_database_id_name_schema ON db_extension(database_id, name, schema);

ALTER SEQUENCE db_extension_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_extension_updated_ts
BEFORE
UPDATE
    ON db_extension FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- vw stores the view for a particular database
-- data is synced periodically from the instance
CREATE TABLE vw (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    definition TEXT NOT NULL,
    comment TEXT NOT NULL
);

CREATE INDEX idx_vw_database_id ON vw(database_id);

CREATE UNIQUE INDEX idx_vw_unique_database_id_name ON vw(database_id, name);

ALTER SEQUENCE vw_id_seq RESTART WITH 101;

CREATE TRIGGER update_vw_updated_ts
BEFORE
UPDATE
    ON vw FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- data_source table stores the data source for a particular database
CREATE TABLE data_source (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('ADMIN', 'RW', 'RO')),
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    ssl_key TEXT NOT NULL DEFAULT '',
    ssl_cert TEXT NOT NULL DEFAULT '',
    ssl_ca TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_data_source_instance_id ON data_source(instance_id);

CREATE UNIQUE INDEX idx_data_source_unique_database_id_name ON data_source(database_id, name);

ALTER SEQUENCE data_source_id_seq RESTART WITH 101;

CREATE TRIGGER update_data_source_updated_ts
BEFORE
UPDATE
    ON data_source FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- backup stores the backups for a particular database.
CREATE TABLE backup (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING_CREATE', 'DONE', 'FAILED')),
    type TEXT NOT NULL CHECK (type IN ('MANUAL', 'AUTOMATIC')),
    storage_backend TEXT NOT NULL CHECK (storage_backend IN ('LOCAL', 'S3', 'GCS', 'OSS')),
    migration_history_version TEXT NOT NULL,
    path TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_backup_database_id ON backup(database_id);

CREATE UNIQUE INDEX idx_backup_unique_database_id_name ON backup(database_id, name);

ALTER SEQUENCE backup_id_seq RESTART WITH 101;

CREATE TRIGGER update_backup_updated_ts
BEFORE
UPDATE
    ON backup FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- backup_setting stores the backup settings for a particular database.
-- This is a strict version of cron expression using UTC timezone uniformly.
CREATE TABLE backup_setting (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    enabled BOOLEAN NOT NULL,
    hour INTEGER NOT NULL CHECK (hour >= 0 AND hour <= 23),
    -- day_of_week can be -1 which is wildcard (daily automatic backup).
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= -1 AND day_of_week <= 6),
    -- hook_url is the callback url to be requested after a successful backup.
    hook_url TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_backup_setting_unique_database_id ON backup_setting(database_id);

ALTER SEQUENCE backup_setting_id_seq RESTART WITH 101;

CREATE TRIGGER update_backup_setting_updated_ts
BEFORE
UPDATE
    ON backup_setting FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-----------------------
-- Pipeline related BEGIN
-- pipeline table
CREATE TABLE pipeline (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'DONE', 'CANCELED'))
);

CREATE INDEX idx_pipeline_status ON pipeline(status);

ALTER SEQUENCE pipeline_id_seq RESTART WITH 101;

CREATE TRIGGER update_pipeline_updated_ts
BEFORE
UPDATE
    ON pipeline FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- stage table stores the stage for the pipeline
CREATE TABLE stage (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL
);

CREATE INDEX idx_stage_pipeline_id ON stage(pipeline_id);

ALTER SEQUENCE stage_id_seq RESTART WITH 101;

CREATE TRIGGER update_stage_updated_ts
BEFORE
UPDATE
    ON stage FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task table stores the task for the stage
CREATE TABLE task (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    stage_id INTEGER NOT NULL REFERENCES stage (id),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- Could be empty for creating database task when the task isn't yet completed successfully.
    database_id INTEGER REFERENCES db (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('PENDING', 'PENDING_APPROVAL', 'RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    type TEXT NOT NULL CHECK (type LIKE 'bb.task.%'),
    payload JSONB NOT NULL DEFAULT '{}',
    earliest_allowed_ts BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_task_pipeline_id_stage_id ON task(pipeline_id, stage_id);

CREATE INDEX idx_task_status ON task(status);

CREATE INDEX idx_task_earliest_allowed_ts ON task(earliest_allowed_ts);

ALTER SEQUENCE task_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_updated_ts
BEFORE
UPDATE
    ON task FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task run table stores the task run
CREATE TABLE task_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    task_id INTEGER NOT NULL REFERENCES task (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    type TEXT NOT NULL CHECK (type LIKE 'bb.task.%'),
    code INTEGER NOT NULL DEFAULT 0,
    comment TEXT NOT NULL DEFAULT '',
    -- result saves the task run result in json format
    result  JSONB NOT NULL DEFAULT '{}',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_run_task_id ON task_run(task_id);

ALTER SEQUENCE task_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_run_updated_ts
BEFORE
UPDATE
    ON task_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- task check run table stores the task check run
CREATE TABLE task_check_run (
    id SERIAL PRIMARY KEY,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    task_id INTEGER NOT NULL REFERENCES task (id),
    status TEXT NOT NULL CHECK (status IN ('RUNNING', 'DONE', 'FAILED', 'CANCELED')),
    type TEXT NOT NULL CHECK (type LIKE 'bb.task-check.%'),
    code INTEGER NOT NULL DEFAULT 0,
    comment TEXT NOT NULL DEFAULT '',
    -- result saves the task check run result in json format
    result  JSONB NOT NULL DEFAULT '{}',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_task_check_run_task_id ON task_check_run(task_id);

ALTER SEQUENCE task_check_run_id_seq RESTART WITH 101;

CREATE TRIGGER update_task_check_run_updated_ts
BEFORE
UPDATE
    ON task_check_run FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Pipeline related END
-----------------------
-- issue
-- Each issue links a pipeline driving the resolution.
CREATE TABLE issue (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('OPEN', 'DONE', 'CANCELED')),
    type TEXT NOT NULL CHECK (type LIKE 'bb.issue.%'),
    description TEXT NOT NULL DEFAULT '',
    -- While changing assignee_id, one should only change it to a non-robot DBA/owner.
    assignee_id INTEGER NOT NULL REFERENCES principal (id),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_issue_project_id ON issue(project_id);

CREATE INDEX idx_issue_pipeline_id ON issue(pipeline_id);

CREATE INDEX idx_issue_creator_id ON issue(creator_id);

CREATE INDEX idx_issue_assignee_id ON issue(assignee_id);

CREATE INDEX idx_issue_created_ts ON issue(created_ts);

ALTER SEQUENCE issue_id_seq RESTART WITH 101;

CREATE TRIGGER update_issue_updated_ts
BEFORE
UPDATE
    ON issue FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- stores the issue subscribers. Unlike other tables, it doesn't have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We use a separate table mainly because we can't leverage indexed query if the subscriber id is stored
-- as a comma separated id list in the issue table.
CREATE TABLE issue_subscriber (
    issue_id INTEGER NOT NULL REFERENCES issue (id),
    subscriber_id INTEGER NOT NULL REFERENCES principal (id),
    PRIMARY KEY (issue_id, subscriber_id)
);

CREATE INDEX idx_issue_subscriber_subscriber_id ON issue_subscriber(subscriber_id);

-- activity table stores the activity for the container such as issue
CREATE TABLE activity (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    container_id INTEGER NOT NULL CHECK (container_id > 0),
    type TEXT NOT NULL CHECK (type LIKE 'bb.%'),
    level TEXT NOT NULL CHECK (level IN ('INFO', 'WARN', 'ERROR')),
    comment TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_activity_container_id ON activity(container_id);

CREATE INDEX idx_activity_created_ts ON activity(created_ts);

ALTER SEQUENCE activity_id_seq RESTART WITH 101;

CREATE TRIGGER update_activity_updated_ts
BEFORE
UPDATE
    ON activity FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- inbox table stores the inbox entry for the corresponding activity.
-- Unlike other tables, it doesn't have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We design in this way because:
-- 1. The table may potentially contain a lot of rows (an issue activity will generate one inbox record per issue subscriber)
-- 2. Does not provide much value besides what's contained in the related activity record.
CREATE TABLE inbox (
    id SERIAL PRIMARY KEY,
    receiver_id INTEGER NOT NULL REFERENCES principal (id),
    activity_id INTEGER NOT NULL REFERENCES activity (id),
    status TEXT NOT NULL CHECK (status IN ('UNREAD', 'READ'))
);

CREATE INDEX idx_inbox_receiver_id_activity_id ON inbox(receiver_id, activity_id);

CREATE INDEX idx_inbox_receiver_id_status ON inbox(receiver_id, status);

ALTER SEQUENCE inbox_id_seq RESTART WITH 101;

-- bookmark table stores the bookmark for the user
CREATE TABLE bookmark (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    link TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_bookmark_unique_creator_id_link ON bookmark(creator_id, link);

ALTER SEQUENCE bookmark_id_seq RESTART WITH 101;

CREATE TRIGGER update_bookmark_updated_ts
BEFORE
UPDATE
    ON bookmark FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- vcs table stores the version control provider config
CREATE TABLE vcs (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('GITLAB_SELF_HOST')),
    instance_url TEXT NOT NULL CHECK ((instance_url LIKE 'http://%' OR instance_url LIKE 'https://%') AND instance_url = rtrim(instance_url, '/')),
    api_url TEXT NOT NULL CHECK ((api_url LIKE 'http://%' OR api_url LIKE 'https://%') AND api_url = rtrim(api_url, '/')),
    application_id TEXT NOT NULL,
    secret TEXT NOT NULL
);

ALTER SEQUENCE vcs_id_seq RESTART WITH 101;

CREATE TRIGGER update_vcs_updated_ts
BEFORE
UPDATE
    ON vcs FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- repository table stores the repository setting for a project
-- A vcs is associated with many repositories.
-- A project can only link one repository (at least for now).
CREATE TABLE repository (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    vcs_id INTEGER NOT NULL REFERENCES vcs (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    -- Name from the corresponding VCS provider.
    -- For GitLab, this is the project name. e.g. project 1
    name TEXT NOT NULL,
    -- Full path from the corresponding VCS provider.
    -- For GitLab, this is the project full path. e.g. group1/project-1
    full_path TEXT NOT NULL,
    -- Web url from the corresponding VCS provider.
    -- For GitLab, this is the project web url. e.g. https://gitlab.example.com/group1/project-1
    web_url TEXT NOT NULL,
    -- Branch we are interested.
    -- For GitLab, this corresponds to webhook's push_events_branch_filter. Wildcard is supported
    branch_filter TEXT NOT NULL DEFAULT '',
    -- Base working directory we are interested.
    base_directory TEXT NOT NULL DEFAULT '',
    -- The file path template for matching the commited migration script.
    file_path_template TEXT NOT NULL DEFAULT '',
    -- The file path template for storing the latest schema auto-generated by Bytebase after migration.
    -- If empty, then Bytebase won't auto generate it.
    schema_path_template TEXT NOT NULL DEFAULT '',
    -- Repository id from the corresponding VCS provider.
    -- For GitLab, this is the project id. e.g. 123
    external_id TEXT NOT NULL,
    -- Push webhook id from the corresponding VCS provider.
    -- For GitLab, this is the project webhook id. e.g. 123
    external_webhook_id TEXT NOT NULL,
    -- Identify the host of the webhook url where the webhook event sends. We store this to identify stale webhook url whose url doesn't match the current bytebase --host.
    webhook_url_host TEXT NOT NULL,
    -- Identify the target repository receiving the webhook event. This is a random string.
    webhook_endpoint_id TEXT NOT NULL,
    -- For GitLab, webhook request contains this in the 'X-Gitlab-Token" header and we compare it with the one stored in db to validate it sends to the expected endpoint.
    webhook_secret_token TEXT NOT NULL,
    -- access_token, expires_ts, refresh_token belongs to the user linking the project to the VCS repository.
    access_token TEXT NOT NULL,
    expires_ts BIGINT NOT NULL,
    refresh_token TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_repository_unique_project_id ON repository(project_id);

CREATE UNIQUE INDEX idx_repository_unique_webhook_endpoint_id ON repository(webhook_endpoint_id);

ALTER SEQUENCE repository_id_seq RESTART WITH 101;

CREATE TRIGGER update_repository_updated_ts
BEFORE
UPDATE
    ON repository FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Anomaly
-- anomaly stores various anomalies found by the scanner.
-- For now, anomaly can be associated with a particular instance or database.
CREATE TABLE anomaly (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- NULL if it's an instance anomaly
    database_id INTEGER NULL REFERENCES db (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.anomaly.%'),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_anomaly_instance_id_row_status_type ON anomaly(instance_id, row_status, type);
CREATE INDEX idx_anomaly_database_id_row_status_type ON anomaly(database_id, row_status, type);

ALTER SEQUENCE anomaly_id_seq RESTART WITH 101;

CREATE TRIGGER update_anomaly_updated_ts
BEFORE
UPDATE
    ON anomaly FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Label
-- label_key stores available label keys at workspace level.
CREATE TABLE label_key (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    key TEXT NOT NULL
);

-- key's are unique within the label_key table.
CREATE UNIQUE INDEX idx_label_key_unique_key ON label_key(key);

ALTER SEQUENCE label_key_id_seq RESTART WITH 101;

CREATE TRIGGER update_label_key_updated_ts
BEFORE
UPDATE
    ON label_key FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- label_value stores available label key values at workspace level.
CREATE TABLE label_value (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    key TEXT NOT NULL REFERENCES label_key(key),
    value TEXT NOT NULL
);

-- key/value's are unique within the label_value table.
CREATE UNIQUE INDEX idx_label_value_unique_key_value ON label_value(key, value);

ALTER SEQUENCE label_value_id_seq RESTART WITH 101;

CREATE TRIGGER update_label_value_updated_ts
BEFORE
UPDATE
    ON label_value FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- db_label stores labels asscociated with databases.
CREATE TABLE db_label (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    database_id INTEGER NOT NULL REFERENCES db (id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY(key, value) REFERENCES label_value(key, value)
);

-- database_id/key's are unique within the db_label table.
CREATE UNIQUE INDEX idx_db_label_unique_database_id_key ON db_label(database_id, key);

ALTER SEQUENCE db_label_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_label_updated_ts
BEFORE
UPDATE
    ON db_label FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- Deployment Configuration.
-- deployment_config stores deployment configurations at project level.
CREATE TABLE deployment_config (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_deployment_config_unique_project_id ON deployment_config(project_id);

ALTER SEQUENCE deployment_config_id_seq RESTART WITH 101;

CREATE TRIGGER update_deployment_config_updated_ts
BEFORE
UPDATE
    ON deployment_config FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

-- sheet table stores general statements.
CREATE TABLE sheet (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id INTEGER NOT NULL REFERENCES project (id),
    database_id INTEGER NULL REFERENCES db (id),
    name TEXT NOT NULL,
    statement TEXT NOT NULL,
    visibility TEXT NOT NULL CHECK (visibility IN ('PRIVATE', 'PROJECT', 'PUBLIC')) DEFAULT 'PRIVATE'
);

CREATE INDEX idx_sheet_creator_id ON sheet(creator_id);

CREATE INDEX idx_sheet_project_id_row_status ON sheet(project_id, row_status);

CREATE INDEX idx_sheet_database_id_row_status ON sheet(database_id, row_status);

ALTER SEQUENCE sheet_id_seq RESTART WITH 101;

CREATE TRIGGER update_sheet_updated_ts
BEFORE
UPDATE
    ON sheet FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
