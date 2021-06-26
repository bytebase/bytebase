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

-- Member
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
    workflow_type TEXT NOT NULL CHECK (workflow_type IN ('UI', 'VCS'))
);

INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        `key`,
        workflow_type
    )
VALUES
    (
        1,
        1,
        1,
        'Default',
        'DEFAULT',
        'UI'
    );

INSERT INTO
    sqlite_sequence (name, seq)
VALUES
    ('project', 100);

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
    `engine` TEXT NOT NULL CHECK (`engine` IN ('MYSQL')),
    host TEXT NOT NULL,
    port TEXT NOT NULL,
    external_link TEXT NOT NULL
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
    name TEXT NOT NULL,
    character_set TEXT NOT NULL,
    `collation` TEXT NOT NULL,
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
    name TEXT NOT NULL,
    `engine` TEXT NOT NULL,
    `collation` TEXT NOT NULL,
    row_count BIGINT NOT NULL,
    data_size BIGINT NOT NULL,
    index_size BIGINT NOT NULL,
    sync_status TEXT NOT NULL CHECK (
        sync_status IN ('OK', 'DRIFTED', 'NOT_FOUND')
    ),
    last_successful_sync_ts BIGINT NOT NULL,
    UNIQUE(database_id, name)
);

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
    UNIQUE(database_id, name)
);

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
    -- Could be empty FOR tasks LIKE creating DATABASE
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
    payload TEXT NOT NULL DEFAULT '{}'
);

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
    error TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT '{}'
);

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
    description TEXT NOT NULL,
    assignee_id INTEGER REFERENCES principal (id),
    subscriber_id_list TEXT NOT NULL,
    payload TEXT NOT NULL DEFAULT '{}'
);

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
    `comment` TEXT NOT NULL DEFAULT '',
    payload TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_activity_container_id ON activity(container_id);

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
            instance_url LIKE 'http:/%'
            OR instance_url LIKE 'https:/%'
        )
        AND instance_url = rtrim(instance_url, '/')
    ),
    api_url TEXT NOT NULL CHECK (
        (
            api_url LIKE 'http:/%'
            OR api_url LIKE 'https:/%'
        )
        AND api_url = rtrim(api_url, '/')
    ),
    application_id TEXT NOT NULL,
    secret TEXT NOT NULL,
    access_token TEXT NOT NULL,
    expires_ts BIGINT NOT NULL,
    refresh_token TEXT NOT NULL
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
    -- Identify the target repository receiving the webhook event. This is a random string, and we don't store the full URL because the backend may change host.
    webhook_endpoint_id TEXT NOT NULL UNIQUE,
    -- For GitLab, webhook request contains this in the 'X-Gitlab-Token" header and we can compare it with the one stored in db to validate it sends to the expected endpoint.
    secret_token TEXT NOT NULL
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