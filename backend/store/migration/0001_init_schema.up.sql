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
    `status` TEXT NOT NULL CHECK (`status` IN ('INVITED', 'ACTIVE')),
    `type` TEXT NOT NULL CHECK (`type` IN ('END_USER', 'SYSTEM_BOT')),
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('principal', 1000);

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
        `status`,
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
        'ACTIVE',
        'SYSTEM_BOT',
        'Bytebase',
        'support@bytebase.com',
        ''
    );

-- workspace
-- We only allow a single workspace for on-premise deployment.
-- So theoretically speaking, we don't need to create a workspace table.
-- But we create this table mostly for preparing a future cloud version
-- where we need to support many workspaces and it would be painful to
-- introduce workspace table at that time.
CREATE TABLE workspace (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('workspace', 1000);

CREATE TRIGGER IF NOT EXISTS `trigger_update_workspace_modification_time`
AFTER
UPDATE
    ON `workspace` FOR EACH ROW BEGIN
UPDATE
    `workspace`
SET
    updated_ts = (strftime('%s', 'now'))
WHERE
    rowid = old.rowid;

END;

-- Default workspace for on-premise deployment, id is 1
INSERT INTO
    workspace (
        id,
        creator_id,
        updater_id,
        slug,
        name
    )
VALUES
    (1, 1, 1, '', 'Default workspace');

-- Member table stores the workspace membership
CREATE TABLE member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    `role` TEXT NOT NULL CHECK (
        `role` IN ('OWNER', 'DBA', 'DEVELOPER')
    ),
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    UNIQUE(workspace_id, principal_id)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('member', 1000);

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

-- Environment table stores the workspace environment
CREATE TABLE environment (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    name TEXT NOT NULL,
    `order` INTEGER NOT NULL,
    UNIQUE(workspace_id, name)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('environment', 1000);

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

-- Project table stores the workspace project
CREATE TABLE project (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    name TEXT NOT NULL,
    `key` TEXT NOT NULL,
    UNIQUE(workspace_id, `key`)
);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `key`
    )
VALUES
    (
        1,
        1,
        1,
        1,
        'Default',
        'DFLT'
    );

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('project', 1000);

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

-- Project member table stores the workspace project membership
CREATE TABLE project_member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    `role` TEXT NOT NULL CHECK (`role` IN ('OWNER', 'DEVELOPER')),
    principal_id INTEGER NOT NULL REFERENCES principal (id),
    UNIQUE(project_id, principal_id)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('project_member', 1000);

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

-- Instance table stores the workspace instance
CREATE TABLE instance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL,
    external_link TEXT NOT NULL,
    host TEXT NOT NULL,
    port TEXT NOT NULL,
    UNIQUE(workspace_id, name)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('instance', 1000);

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

-- db table stores the databases for a particular instance
-- data is sycned periodically from the instance
CREATE TABLE db (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    name TEXT NOT NULL,
    sync_status TEXT NOT NULL CHECK (
        sync_status IN ('OK', 'DRIFTED', 'NOT_FOUND')
    ),
    last_successful_sync_ts BIGINT NOT NULL,
    fingerprint TEXT NOT NULL,
    UNIQUE(instance_id, name)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('db', 1000);

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
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    database_id INTEGER NOT NULL REFERENCES db (id),
    name TEXT NOT NULL,
    `type` TEXT NOT NULL CHECK (TYPE IN ('ADMIN', 'RW', 'RO')),
    username TEXT NOT NULL,
    `password` TEXT NOT NULL,
    UNIQUE(instance_id, name)
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('data_source', 1000);

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

-----------------------
-- Pipeline related BEGIN
-- pipeline table stores the workspace pipeline 
CREATE TABLE pipeline (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    row_status TEXT NOT NULL CHECK (
        row_status IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ) DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (`status` IN ('OPEN', 'DONE', 'CANCELED'))
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('pipeline', 1000);

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
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    environment_id INTEGER NOT NULL REFERENCES environment (id),
    name TEXT NOT NULL,
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.stage.%')
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('stage', 1000);

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
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    stage_id INTEGER NOT NULL REFERENCES stage (id),
    -- Could be empty for tasks like creating database
    database_id INTEGER REFERENCES db (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (
        `status` IN (
            'PENDING',
            'RUNNING',
            'DONE',
            'FAILED',
            "SKIPPED"
        )
    ),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.task.%'),
    `when` TEXT NOT NULL CHECK (`when` IN ('ON_SUCCESS', 'MANUAL')),
    payload TEXT NOT NULL DEFAULT ''
);

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('task', 1000);

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

-- Pipeline related END
-----------------------
-- issue table stores the workspace issue
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
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    project_id INTEGER NOT NULL REFERENCES project (id),
    pipeline_id INTEGER NOT NULL REFERENCES pipeline (id),
    name TEXT NOT NULL,
    `status` TEXT NOT NULL CHECK (`status` IN ('OPEN', 'DONE', 'CANCELED')),
    `type` TEXT NOT NULL CHECK (`type` LIKE 'bb.%'),
    description TEXT NOT NULL,
    assignee_id INTEGER REFERENCES principal (id),
    subscriber_id_list TEXT NOT NULL,
    `sql` TEXT NOT NULL,
    rollback_sql TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT ''
);