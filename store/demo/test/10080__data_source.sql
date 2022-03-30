INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8001,
        1,
        1,
        6001,
        7001,
        'Admin data source',
        'ADMIN',
        'root',
        'testpwd1'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8002,
        1,
        1,
        6002,
        7005,
        'Admin data source',
        'ADMIN',
        'admin',
        'Integration12345'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8003,
        1,
        1,
        6003,
        7009,
        'Admin data source',
        'ADMIN',
        'admin',
        'Staging12345'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8004,
        1,
        1,
        6004,
        7013,
        'Admin data source',
        'ADMIN',
        'root',
        'testpwd1'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8005,
        1,
        1,
        6005,
        7017,
        'Admin data source',
        'ADMIN',
        'postgres',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8006,
        1,
        1,
        6006,
        7018,
        'Admin data source',
        'ADMIN',
        'postgre_demo_user',
        '3QixNmRMGhklX6B1lmCZ3ZsHFPIE5EgG'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8007,
        1,
        1,
        6007,
        7020,
        'Admin data source',
        'ADMIN',
        'root',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8008,
        1,
        1,
        6008,
        7021,
        'Admin data source',
        'ADMIN',
        '',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8009,
        1,
        1,
        6009,
        7022,
        'Admin data source',
        'ADMIN',
        '',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        PASSWORD
    )
VALUES
    (
        8010,
        1,
        1,
        6010,
        7031,
        'Admin data source',
        'ADMIN',
        'admin',
        'Prod12345'
    );

ALTER SEQUENCE data_source_id_seq RESTART WITH 8011;
