-- Database for sandbox environment instance 6001
INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7002,
        101,
        101,
        6001,
        3001,
        'testdb_dev',
        'OK',
        1624558090,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7003,
        101,
        101,
        6001,
        3002,
        'shop',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7004,
        101,
        101,
        6001,
        3003,
        'blog',
        'OK',
        0,
        ''
    );

-- Database for integration environment instance 6002
INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7006,
        101,
        101,
        6002,
        3001,
        'testdb_integration',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7007,
        101,
        101,
        6002,
        3002,
        'shop',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7008,
        101,
        101,
        6002,
        3003,
        'blog',
        'OK',
        0,
        ''
    );

-- Database for staging environment instance 6003
INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7010,
        101,
        101,
        6003,
        3001,
        'testdb_staging',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7011,
        101,
        101,
        6003,
        3002,
        'shop',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7012,
        101,
        101,
        6003,
        3003,
        'blog',
        'OK',
        0,
        ''
    );

-- Database for prod environment instance 6004
INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7014,
        101,
        101,
        6004,
        3001,
        'testdb_prod',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7015,
        101,
        101,
        6004,
        3002,
        'shop',
        'OK',
        0,
        ''
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        7016,
        101,
        101,
        6004,
        3003,
        'blog',
        'OK',
        0,
        ''
    );


ALTER SEQUENCE db_id_seq RESTART WITH 7023;
