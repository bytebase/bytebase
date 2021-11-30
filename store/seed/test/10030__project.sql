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
        3001,
        101,
        101,
        'Test (UI)',
        'TEST',
        'UI',
        'PUBLIC'
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
        3002,
        101,
        101,
        'Shop (Git)',
        'SHP',
        'VCS',
        'PUBLIC'
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
        3003,
        101,
        101,
        'Blog (Git)',
        'BLG',
        'VCS',
        'PUBLIC'
    );

INSERT INTO
    project (
        id,
        row_status,
        creator_id,
        updater_id,
        name,
        `key`,
        workflow_type,
        visibility
    )
VALUES
    (
        3004,
        'ARCHIVED',
        101,
        101,
        'Retired Project',
        'RTR',
        'UI',
        'PUBLIC'
    );
