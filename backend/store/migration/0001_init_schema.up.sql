CREATE TABLE principal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
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

-- We only allow a single workspace for on-premise deployment.
-- So theoretically speaking, we don't need to create a workspace table.
-- But we create this table mostly for preparing a future cloud version
-- where we need to support many workspaces and it would be painful to
-- introduce workspace table at that time.
CREATE TABLE workspace (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    row_status TEXT NOT NULL CHECK (
        `row_status` IN ('NORMAL', 'ARCHIVED', 'PENDING_DELETE')
    ),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

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
        row_status,
        slug,
        name
    )
VALUES
    (1, 1, 1, 'NORMAL', '', 'Default workspace');

-- Member table stores the workspace membership
CREATE TABLE member (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspace (id),
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    `role` TEXT NOT NULL CHECK (
        `role` IN ('OWNER', 'DBA', 'DEVELOPER', "GUEST")
    ),
    principal_id INTEGER NOT NULL REFERENCES principal (id)
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