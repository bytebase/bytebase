PRAGMA user_version = 10001;

-- principal
CREATE TABLE principal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL UNIQUE,
    `order` INTEGER NOT NULL,
    approval_policy TEXT NOT NULL CHECK (
        approval_policy IN (
            'MANUAL_APPROVAL_NEVER',
            'MANUAL_APPROVAL_ALWAYS'
        )
    )
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

-- Project
CREATE TABLE project (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    name TEXT NOT NULL,
    `key` TEXT NOT NULL UNIQUE,
    workflow_type TEXT NOT NULL CHECK (workflow_type IN ('UI', 'VCS')),
    visibility TEXT NOT NULL CHECK (visibility IN ('PUBLIC', 'PRIVATE'))
);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        `key`,
        workflow_type,
        visibility
    )
VALUES
    (
        1,
        1,
        1,
        'Default',
        'DEFAULT',
        'UI',
        'PUBLIC'
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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

-- Instance
CREATE TABLE instance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL,
    `engine` TEXT NOT NULL CHECK (`engine` IN ('MYSQL', 'POSTGRES')),
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    sync_status TEXT NOT NULL CHECK (sync_status IN ('OK', 'NOT_FOUND')),
    last_successful_sync_ts BIGINT NOT NULL,
    name TEXT NOT NULL,
    character_set TEXT NOT NULL,
    `collation` TEXT NOT NULL,
    UNIQUE(instance_id, name)
);

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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    sync_status TEXT NOT NULL CHECK (sync_status IN ('OK', 'NOT_FOUND')),
    last_successful_sync_ts BIGINT NOT NULL,
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    table_id INTEGER NOT NULL REFERENCES tbl (id),
    sync_status TEXT NOT NULL CHECK (sync_status IN ('OK', 'NOT_FOUND')),
    last_successful_sync_ts BIGINT NOT NULL,
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    table_id INTEGER NOT NULL REFERENCES tbl (id),
    sync_status TEXT NOT NULL CHECK (sync_status IN ('OK', 'NOT_FOUND')),
    last_successful_sync_ts BIGINT NOT NULL,
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

-- data_source table stores the data source for a particular database
CREATE TABLE data_source (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (`status` IN ('PENDING_CREATE', 'DONE')),
    `type` TEXT NOT NULL CHECK (`type` IN ('MANUAL', 'AUTOMATIC')),
    storage_backend TEXT NOT NULL CHECK (storage_backend IN ('LOCAL')),
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

-----------------------
-- Pipeline related BEGIN
-- pipeline table
CREATE TABLE pipeline (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
    payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_task_pipeline_id_stage_id ON task(pipeline_id, stage_id);

CREATE INDEX idx_task_status ON task(`status`);

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
    detail TEXT NOT NULL DEFAULT '',
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
    `task`
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    container_id INTEGER NOT NULL CHECK (container_id != 0),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.%'),
    `level` TEXT NOT NULL CHECK (`level` IN ('INFO', 'WARNING', 'ERROR')),
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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

-- repo table stores the repository setting for a project
-- A vcs is associated with many repositories.
-- A project can only link one repository (at least for now).
CREATE TABLE repo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
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
    -- Base working directory we are interested.
    base_directory TEXT NOT NULL DEFAULT '',
    -- Branch we are interested.
    -- For GitLab, this corresponds to webhook's push_events_branch_filter. Wildcard is supported
    branch_filter TEXT NOT NULL CHECK (trim(branch_filter) != ''),
    -- Repo id from the corresponding VCS provider.
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
    ('repo', 100);

CREATE TRIGGER IF NOT EXISTS `trigger_update_repo_modification_time`
AFTER
UPDATE
    ON `repo` FOR EACH ROW BEGIN
UPDATE
    `repo`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;