CREATE TABLE principal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT (strftime('%s', 'now')),
    -- "INVITED" | "ACTIVE"
    `status` TEXT NOT NULL,
    -- "END_USER" | "SYSTEM_BOT"
    `type` TEXT NOT NULL,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);

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
    -- "NORMAL" | "ARCHIVED" | "PENDING_DELETE"
    row_status TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL
);

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