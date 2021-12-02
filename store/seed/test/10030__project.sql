INSERT INTO
    project (
        id,
        creator_id,
        updater_id,
        name,
        `key`,
        workflow_type,
        visibility,
        tenant_mode
    )
VALUES
    (
        3001,
        101,
        101,
        'Test (UI)',
        'TEST',
        'UI',
        'PUBLIC',
        'DISABLED'
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
        tenant_mode
    )
VALUES
    (
        3002,
        101,
        101,
        'Shop (Git)',
        'SHP',
        'VCS',
        'PUBLIC',
        'DISABLED'
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
        tenant_mode
    )
VALUES
    (
        3003,
        101,
        101,
        'Blog (Git)',
        'BLG',
        'VCS',
        'PUBLIC',
        'DISABLED'
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
        visibility,
        tenant_mode
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
        'PUBLIC',
        'DISABLED'
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
        tenant_mode
    )
VALUES
    (
        3005,
        101,
        101,
        'Tenant (Git)',
        'TNTG',
        'VCS',
        'PUBLIC',
        'TENANT'
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
        tenant_mode
    )
VALUES
    (
        3006,
        101,
        101,
        'Tenant (UI)',
        'TNTU',
        'UI',
        'PUBLIC',
        'TENANT'
    );
