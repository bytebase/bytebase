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
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
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
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
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