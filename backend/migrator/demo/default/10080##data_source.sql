INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8001,
        1,
        1,
        6001,
        'Admin data source',
        'ADMIN',
        '127.0.0.1',
        '3306',
        'root',
        'testpwd1'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8002,
        1,
        1,
        6002,
        'Admin data source',
        'ADMIN',
        'mysql.integration.example.com',
        '3306',
        'admin',
        'Integration12345'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8003,
        1,
        1,
        6003,
        'Admin data source',
        'ADMIN',
        'mysql.staging.example.com',
        '3306',
        'admin',
        'Staging12345'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8004,
        1,
        1,
        6004,
        'Admin data source',
        'ADMIN',
        'mysql.prod.example.com',
        '3306',
        'root',
        'testpwd1'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8005,
        1,
        1,
        6005,
        'Admin data source',
        'ADMIN',
        '127.0.0.1',
        '5432',
        'postgres',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8006,
        1,
        1,
        6006,
        'Admin data source',
        'ADMIN',
        'dpg-c8a7pcd0mal7gtod05p0',
        '5432',
        'postgre_demo_user',
        '3QixNmRMGhklX6B1lmCZ3ZsHFPIE5EgG'
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8007,
        1,
        1,
        6007,
        'Admin data source',
        'ADMIN',
        '127.0.0.1',
        '4000',
        'root',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8008,
        1,
        1,
        6008,
        'Admin data source',
        'ADMIN',
        '127.0.0.1',
        '9000',
        '',
        ''
    );

INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        name,
        TYPE,
        host,
        port,
        username,
        PASSWORD
    )
VALUES
    (
        8009,
        1,
        1,
        6009,
        'Admin data source',
        'ADMIN',
        '127.0.0.1',
        '',
        '',
        ''
    );

ALTER SEQUENCE data_source_id_seq RESTART WITH 8010;
