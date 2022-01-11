PRAGMA user_version = 10001;
PRAGMA foreign_keys = ON;

-- principal
CREATE TABLE principal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    `type` TEXT NOT NULL CHECK (`type` IN ('END_USER', 'SYSTEM_BOT')),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

CREATE INDEX idx_principal_email ON principal(email);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('principal', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_principal_modification_time`
AFTER
UPDATE
    ON `principal` FOR EACH ROW BEGIN
UPDATE
    `principal`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Default bytebase system account id is 1
INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `type`,
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

-- Setting
CREATE TABLE setting (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT ''
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('setting', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_setting_modification_time`
AFTER
UPDATE
    ON `setting` FOR EACH ROW BEGIN
UPDATE
    `setting`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Member
-- We separate the concept from Principal because if we support multiple workspace in the future, each workspace can have different member for the same principal
CREATE TABLE member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    `status` TEXT NOT NULL CHECK (`status` IN ('INVITED', 'ACTIVE')),
    `role` TEXT NOT NULL CHECK (
        `role` IN ('OWNER', 'DBA', 'DEVELOPER')
    ),
    principal_id INTEGER NOT NULL REFERENCES principal (id) UNIQUE
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('member', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_member_modification_time`
AFTER
UPDATE
    ON `member` FOR EACH ROW BEGIN
UPDATE
    `member`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Environment
CREATE TABLE environment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL UNIQUE,
    `order` INTEGER NOT NULL
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('environment', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_environment_modification_time`
AFTER
UPDATE
    ON `environment` FOR EACH ROW BEGIN
UPDATE
    `environment`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Policy
-- policy stores the policies for each environment.
-- Policies are associated with environments. Since we may have policies not associated with environment later, we name the table policy.
CREATE TABLE policy (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.policy.%'),
    payload TEXT NOT NULL
);

CREATE INDEX idx_policy_environment_id ON policy(environment_id);

CREATE UNIQUE INDEX idx_policy_environment_id_type ON policy(environment_id, type);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('policy', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_policy_modification_time`
AFTER
UPDATE
    ON `policy` FOR EACH ROW BEGIN
UPDATE
    `policy`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Project
CREATE TABLE project (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL,
    `key` TEXT NOT NULL UNIQUE,
    workflow_type TEXT NOT NULL CHECK (workflow_type IN ('UI', 'VCS')),
    visibility TEXT NOT NULL CHECK (visibility IN ('PUBLIC', 'PRIVATE')),
    tenant_mode TEXT NOT NULL DEFAULT 'DISABLED' CHECK (tenant_mode IN ('DISABLED', 'TENANT')),
    -- db_name_template is only used when a project is in tenant mode.
    -- Empty value means {{DB_NAME}}.
    db_name_template TEXT NOT NULL
);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        `key`,
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

UPDATE
    sqlite_sequence
SET
    seq = 100
WHERE
    name = 'project';

CREATE TRIGGER IF NOT EXISTS `trigger_update_project_modification_time`
AFTER
UPDATE
    ON `project` FOR EACH ROW BEGIN
UPDATE
    `project`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Project member
CREATE TABLE project_member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    project_id INTEGER NOT NULL REFERENCES project (id),
    `role` TEXT NOT NULL CHECK (`role` IN ('OWNER', 'DEVELOPER')),
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    UNIQUE(project_id, principal_id)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('project_member', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_project_member_modification_time`
AFTER
UPDATE
    ON `project_member` FOR EACH ROW BEGIN
UPDATE
    `project_member`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Project Hook
CREATE TABLE project_webhook (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    project_id INTEGER NOT NULL REFERENCES project (id),
    type TEXT NOT NULL CHECK (type LIKE 'bb.plugin.webhook.%'),
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    -- Comma separated list of activity triggers.
    activity_list TEXT NOT NULL,
    UNIQUE(project_id, url)
);

CREATE INDEX idx_project_webhook_project_id ON project_webhook(project_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('project_webhook', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_project_webhook_modification_time`
AFTER
UPDATE
    ON `project_webhook` FOR EACH ROW BEGIN
UPDATE
    `project_webhook`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Instance
CREATE TABLE instance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL,
    `engine` TEXT NOT NULL CHECK (`engine` IN ('MYSQL', 'POSTGRES', 'TIDB', 'CLICKHOUSE', 'SNOWFLAKE', 'SQLITE')),
    engine_version TEXT NOT NULL DEFAULT '',
    host TEXT NOT NULL,
    port TEXT NOT NULL,
    external_link TEXT NOT NULL DEFAULT ''
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('instance', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_instance_modification_time`
AFTER
UPDATE
    ON `instance` FOR EACH ROW BEGIN
UPDATE
    `instance`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Instance user stores the users for a particular instance
CREATE TABLE instance_user (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    name TEXT NOT NULL,
    grant TEXT NOT NULL,
    UNIQUE(instance_id, name)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('instance_user', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_instance_user_modification_time`
AFTER
UPDATE
    ON `instance_user` FOR EACH ROW BEGIN
UPDATE
    `instance_user`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- db stores the databases for a particular instance
-- data is synced periodically from the instance
CREATE TABLE db (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    -- If db is restored from a backup, then we will record that backup id. We can thus trace up to the original db.
    source_backup_id INTEGER REFERENCES backup (id) ON DELETE SET NULL,
    sync_status TEXT NOT NULL CHECK (sync_status IN ('OK', 'NOT_FOUND')),
    last_successful_sync_ts BIGINT NOT NULL,
    schema_version TEXT NOT NULL,
    name TEXT NOT NULL,
    character_set TEXT NOT NULL,
    `collation` TEXT NOT NULL,
    UNIQUE(instance_id, name)
);

CREATE INDEX idx_db_instance_id ON db(instance_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('db', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_db_modification_time`
AFTER
UPDATE
    ON `db` FOR EACH ROW BEGIN
UPDATE
    `db`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- tbl stores the table for a particular database
-- data is synced periodically from the instance
CREATE TABLE tbl (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    `type` TEXT NOT NULL,
    `engine` TEXT NOT NULL,
    `collation` TEXT NOT NULL,
    row_count BIGINT NOT NULL,
    data_size BIGINT NOT NULL,
    index_size BIGINT NOT NULL,
    data_free BIGINT NOT NULL,
    create_options TEXT NOT NULL,
    `comment` TEXT NOT NULL,
    UNIQUE(database_id, name)
);

CREATE INDEX idx_tbl_database_id ON tbl(database_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('tbl', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_tbl_modification_time`
AFTER
UPDATE
    ON `tbl` FOR EACH ROW BEGIN
UPDATE
    `tbl`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- col stores the column for a particular table from a particular database
-- data is synced periodically from the instance
CREATE TABLE col (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    table_id INTEGER NOT NULL REFERENCES tbl (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    position INTEGER NOT NULL,
    `default` TEXT,
    `nullable` INTEGER NOT NULL,
    `type` TEXT NOT NULL,
    character_set TEXT NOT NULL,
    `collation` TEXT NOT NULL,
    `comment` TEXT NOT NULL,
    UNIQUE(database_id, table_id, name)
);

CREATE INDEX idx_col_database_id_table_id ON col(database_id, table_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('col', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_col_modification_time`
AFTER
UPDATE
    ON `col` FOR EACH ROW BEGIN
UPDATE
    `col`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- idx stores the index for a particular table from a particular database
-- data is synced periodically from the instance
CREATE TABLE idx (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    table_id INTEGER NOT NULL REFERENCES tbl (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    expression TEXT NOT NULL,
    position INTEGER NOT NULL,
    `type` TEXT NOT NULL,
    `unique` INTEGER NOT NULL,
    visible INTEGER NOT NULL,
    `comment` TEXT NOT NULL,
    UNIQUE(database_id, table_id, name, expression)
);

CREATE INDEX idx_idx_database_id_table_id ON idx(database_id, table_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('idx', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_idx_modification_time`
AFTER
UPDATE
    ON `idx` FOR EACH ROW BEGIN
UPDATE
    `idx`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- vw stores the view for a particular database
-- data is synced periodically from the instance
CREATE TABLE vw (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    definition TEXT NOT NULL,
    comment TEXT NOT NULL,
    UNIQUE(database_id, name)
);

CREATE INDEX idx_vw_database_id ON vw(database_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('vw', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_vw_modification_time`
AFTER
UPDATE
    ON `vw` FOR EACH ROW BEGIN
UPDATE
    `vw`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- data_source table stores the data source for a particular database
CREATE TABLE data_source (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    `type` TEXT NOT NULL CHECK (TYPE IN ('ADMIN', 'RW', 'RO')),
    username TEXT NOT NULL,
    `password` TEXT NOT NULL,
    ssl_key TEXT NOT NULL DEFAULT '',
    ssl_cert TEXT NOT NULL DEFAULT '',
    ssl_ca TEXT NOT NULL DEFAULT '',
    UNIQUE(database_id, name)
);

CREATE INDEX idx_data_source_instance_id ON data_source(instance_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('data_source', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_data_source_modification_time`
AFTER
UPDATE
    ON `data_source` FOR EACH ROW BEGIN
UPDATE
    `data_source`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- backup stores the backups for a particular database.
CREATE TABLE backup (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (`status` IN ('PENDING_CREATE', 'DONE', 'FAILED')),
    `type` TEXT NOT NULL CHECK (`type` IN ('MANUAL', 'AUTOMATIC')),
    storage_backend TEXT NOT NULL CHECK (storage_backend IN ('LOCAL')),
    migration_history_version TEXT NOT NULL,
    path TEXT NOT NULL,
    `comment` TEXT NOT NULL DEFAULT '',
    UNIQUE(database_id, name)
);

CREATE INDEX idx_backup_database_id ON backup(database_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('backup', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_backup_modification_time`
AFTER
UPDATE
    ON `backup` FOR EACH ROW BEGIN
UPDATE
    `backup`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- backup_setting stores the backup settings for a particular database.
-- This is a strict version of cron expression using UTC timezone uniformly.
CREATE TABLE backup_setting (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL UNIQUE REFERENCES db (id),
    `enabled` INTEGER NOT NULL CHECK (`enabled` IN (0, 1)),
    hour INTEGER NOT NULL CHECK (
        0 <= hour
        AND hour < 24
    ),
    -- day_of_week can be -1 which is wildcard (daily automatic backup).
    day_of_week INTEGER NOT NULL CHECK (
        -1 <= day_of_week
        AND day_of_week < 7
    ),
    -- hook_url is the callback url to be requested after a successful backup.
    hook_url TEXT NOT NULL
);

CREATE INDEX idx_backup_setting_database_id ON backup_setting(database_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('backup_setting', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_backup_setting_modification_time`
AFTER
UPDATE
    ON `backup_setting` FOR EACH ROW BEGIN
UPDATE
    `backup_setting`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-----------------------
-- Pipeline related BEGIN
-- pipeline table
CREATE TABLE pipeline (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (`status` IN ('OPEN', 'DONE', 'CANCELED'))
);

CREATE INDEX idx_pipeline_status ON pipeline(`status`);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('pipeline', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_pipeline_modification_time`
AFTER
UPDATE
    ON `pipeline` FOR EACH ROW BEGIN
UPDATE
    `pipeline`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- stage table stores the stage for the pipeline
CREATE TABLE stage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL
);

CREATE INDEX idx_stage_pipeline_id ON stage(pipeline_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('stage', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_stage_modification_time`
AFTER
UPDATE
    ON `stage` FOR EACH ROW BEGIN
UPDATE
    `stage`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- task table stores the task for the stage
CREATE TABLE task (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    stage_id INTEGER NOT NULL REFERENCES stage (id),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- Could be empty for tasks like creating database
    database_id INTEGER REFERENCES db (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (
        `status` IN (
            'PENDING',
            'PENDING_APPROVAL',
            'RUNNING',
            'DONE',
            'FAILED',
            "CANCELED"
        )
    ),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.task.%'),
    payload TEXT NOT NULL DEFAULT '',
    `earliest_allowed_ts` BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_task_pipeline_id_stage_id ON task(pipeline_id, stage_id);

CREATE INDEX idx_task_status ON task(`status`);

CREATE INDEX idx_task_earliest_allowed_ts ON task(earliest_allowed_ts);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('task', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_task_modification_time`
AFTER
UPDATE
    ON `task` FOR EACH ROW BEGIN
UPDATE
    `task`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- task run table stores the task run
CREATE TABLE task_run (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    task_id INTEGER NOT NULL REFERENCES task (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (
        `status` IN (
            'RUNNING',
            'DONE',
            'FAILED',
            "CANCELED"
        )
    ),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.task.%'),
    code INTEGER NOT NULL DEFAULT 0,
    comment TEXT NOT NULL DEFAULT '',
    -- result saves the task run result in json format
    result  TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_task_run_task_id ON task_run(task_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('task_run', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_task_run_modification_time`
AFTER
UPDATE
    ON `task_run` FOR EACH ROW BEGIN
UPDATE
    `task_run`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- task check run table stores the task check run
CREATE TABLE task_check_run (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    task_id INTEGER NOT NULL REFERENCES task (id),
    `status` TEXT NOT NULL CHECK (
        `status` IN (
            'RUNNING',
            'DONE',
            'FAILED',
            "CANCELED"
        )
    ),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.task-check.%'),
    code INTEGER NOT NULL DEFAULT 0,
    comment TEXT NOT NULL DEFAULT '',
    -- result saves the task check run result in json format
    result  TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_task_check_run_task_id ON task_check_run(task_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('task_check_run', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_task_check_run_modification_time`
AFTER
UPDATE
    ON `task_check_run` FOR EACH ROW BEGIN
UPDATE
    `task_check_run`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Pipeline related END
-----------------------
-- issue
-- Each issue links a pipeline driving the resolution.
CREATE TABLE issue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    project_id INTEGER NOT NULL REFERENCES project (id),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (`status` IN ('OPEN', 'DONE', 'CANCELED')),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.issue.%'),
    description TEXT NOT NULL DEFAULT '',
    -- we require an assignee, if user wants to unassign herself, she can re-assign to the system account.
    assignee_id INTEGER NOT NULL REFERENCES principal (id),
    payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_issue_project_id ON issue(project_id);

CREATE INDEX idx_issue_pipeline_id ON issue(pipeline_id);

CREATE INDEX idx_issue_creator_id ON issue(creator_id);

CREATE INDEX idx_issue_assignee_id ON issue(assignee_id);

CREATE INDEX idx_issue_created_ts ON issue(created_ts);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('issue', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_issue_modification_time`
AFTER
UPDATE
    ON `issue` FOR EACH ROW BEGIN
UPDATE
    `issue`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

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
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    container_id INTEGER NOT NULL CHECK (container_id != 0),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.%'),
    `level` TEXT NOT NULL CHECK (`level` IN ('INFO', 'WARN', 'ERROR')),
    `comment` TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_activity_container_id ON activity(container_id);

CREATE INDEX idx_activity_created_ts ON activity(created_ts);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('activity', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_activity_modification_time`
AFTER
UPDATE
    ON `activity` FOR EACH ROW BEGIN
UPDATE
    `activity`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- inbox table stores the inbox entry for the corresponding activity.
-- Unlike other tables, it doesn't have row_status/creator_id/created_ts/updater_id/updated_ts.
-- We design in this way because:
-- 1. The table may potentially contain a lot of rows (an issue activity will generate one inbox record per issue subscriber)
-- 2. Does not provide much value besides what's contained in the related activity record.
CREATE TABLE inbox (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    receiver_id INTEGER NOT NULL REFERENCES principal (id),
    activity_id INTEGER NOT NULL REFERENCES activity (id),
    `status` TEXT NOT NULL CHECK (`status` IN ('UNREAD', 'READ'))
);

CREATE INDEX idx_inbox_receiver_id_activity_id ON inbox(receiver_id, activity_id);

CREATE INDEX idx_inbox_receiver_id_status ON inbox(receiver_id, `status`);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('inbox', 100);

-- bookmark table stores the bookmark for the user
CREATE TABLE bookmark (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL,
    link TEXT NOT NULL,
    UNIQUE(creator_id, link)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('bookmark', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_bookmark_modification_time`
AFTER
UPDATE
    ON `bookmark` FOR EACH ROW BEGIN
UPDATE
    `bookmark`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- vcs table stores the version control provider config
CREATE TABLE vcs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL,
    `type` TEXT NOT NULL CHECK (`type` IN ('GITLAB_SELF_HOST')),
    instance_url TEXT NOT NULL CHECK (
        (
            instance_url LIKE 'http://%'
            OR instance_url LIKE 'https://%'
        )
        AND instance_url = rtrim(instance_url, '/')
    ),
    api_url TEXT NOT NULL CHECK (
        (
            api_url LIKE 'http://%'
            OR api_url LIKE 'https://%'
        )
        AND api_url = rtrim(api_url, '/')
    ),
    application_id TEXT NOT NULL,
    secret TEXT NOT NULL
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('vcs', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_vcs_modification_time`
AFTER
UPDATE
    ON `vcs` FOR EACH ROW BEGIN
UPDATE
    `vcs`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- repository table stores the repository setting for a project
-- A vcs is associated with many repositories.
-- A project can only link one repository (at least for now).
CREATE TABLE repository (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    vcs_id INTEGER NOT NULL REFERENCES vcs (id),
    project_id INTEGER NOT NULL UNIQUE REFERENCES project (id),
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
    branch_filter TEXT NOT NULL CHECK (trim(branch_filter) != ''),
    -- Base working directory we are interested.
    base_directory TEXT NOT NULL DEFAULT '',
    -- The file path template for matching the commited migration script.
    file_path_template TEXT NOT NULL,
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
    webhook_endpoint_id TEXT NOT NULL UNIQUE,
    -- For GitLab, webhook request contains this in the 'X-Gitlab-Token" header and we compare it with the one stored in db to validate it sends to the expected endpoint.
    webhook_secret_token TEXT NOT NULL,
    -- access_token, expires_ts, refresh_token belongs to the user linking the project to the VCS repository.
    access_token TEXT NOT NULL,
    expires_ts BIGINT NOT NULL,
    refresh_token TEXT NOT NULL
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('repository', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_repository_modification_time`
AFTER
UPDATE
    ON `repository` FOR EACH ROW BEGIN
UPDATE
    `repository`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Anomaly
-- anomaly stores various anomalies found by the scanner.
-- For now, anomaly can be associated with a particular instance or database.
CREATE TABLE anomaly (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    -- NULL if it's an instance anomaly
    database_id INTEGER NULL REFERENCES db (id),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.anomaly.%'),
    payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_anomaly_instance_id_row_status_type ON anomaly(instance_id, row_status, type);
CREATE INDEX idx_anomaly_database_id_row_status_type ON anomaly(database_id, row_status, type);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('anomaly', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_anomaly_modification_time`
AFTER
UPDATE
    ON `anomaly` FOR EACH ROW BEGIN
UPDATE
    `anomaly`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Label
-- label_key stores available label keys at workspace level.
CREATE TABLE label_key (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    key TEXT NOT NULL
);

-- key's are unique within the label_key table.
CREATE UNIQUE INDEX idx_label_key_key ON label_key(key);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('label_key', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_label_key_modification_time`
AFTER
UPDATE
    ON `label_key` FOR EACH ROW BEGIN
UPDATE
    `label_key`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- label_value stores available label key values at workspace level.
CREATE TABLE label_value (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY(key) REFERENCES label_key(key)
);

-- key/value's are unique within the label_value table.
CREATE UNIQUE INDEX idx_label_value_key_value ON label_value(key, value);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('label_value', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_label_value_modification_time`
AFTER
UPDATE
    ON `label_value` FOR EACH ROW BEGIN
UPDATE
    `label_value`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- db_label stores labels asscociated with databases.
CREATE TABLE db_label (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    FOREIGN KEY(key, value) REFERENCES label_value(key, value)
);

-- database_id/key's are unique within the db_label table.
CREATE UNIQUE INDEX idx_db_label_database_id_key ON db_label(database_id, key);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('db_label', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_db_label_modification_time`
AFTER
UPDATE
    ON `db_label` FOR EACH ROW BEGIN
UPDATE
    `db_label`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Deployment Configuration.
-- deployment_config stores deployment configurations at project level.
CREATE TABLE deployment_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    config TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_deployment_config_project_id ON deployment_config(project_id);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('deployment_config', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_deployment_config_modification_time`
AFTER
UPDATE
    ON `deployment_config` FOR EACH ROW BEGIN
UPDATE
    `deployment_config`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;
