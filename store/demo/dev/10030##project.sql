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
        3001,
        101,
        101,
        'Test (UI)',
        'TEST',
        'UI',
        'PUBLIC',
        'DISABLED',
        ''
    );

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
        3002,
        101,
        101,
        'Shop (Git)',
        'SHP',
        'VCS',
        'PUBLIC',
        'DISABLED',
        ''
    );

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
        3003,
        101,
        101,
        'Blog (Git)',
        'BLG',
        'VCS',
        'PUBLIC',
        'DISABLED',
        ''
    );

INSERT INTO
    project (
        id,
        row_status,
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
        3004,
        'ARCHIVED',
        101,
        101,
        'Retired Project',
        'RTR',
        'UI',
        'PUBLIC',
        'DISABLED',
        ''
    );

ALTER SEQUENCE principal_id_seq RESTART WITH 3005;
