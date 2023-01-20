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
        db_name_template,
        resource_id
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
        '',
        'test-ui'
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
        db_name_template,
        resource_id
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
        '',
        'shop-git'
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
        db_name_template,
        resource_id
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
        '',
        'blog-git'
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
        db_name_template,
        resource_id
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
        '',
        'retired-project'
    );

ALTER SEQUENCE principal_id_seq RESTART WITH 3005;
