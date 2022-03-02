INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6001,
        101,
        101,
        5001,
        'Localhost Dev MySQL',
        'MYSQL',
        '8.0.19',
        '127.0.0.1',
        '3306',
        'bytebase.com/database/mysql'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6002,
        101,
        101,
        5002,
        'On-premises Integration MySQL',
        'MYSQL',
        '8.0.19',
        'mysql.integration.example.com',
        '3306',
        'bytebase.com/database/mysql'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6003,
        101,
        101,
        5003,
        'On-premises Staging MySQL',
        'MYSQL',
        '8.0.19',
        'mysql.staging.example.com',
        '3306',
        'bytebase.com/database/mysql'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6004,
        101,
        101,
        5004,
        'On-premises Prod MySQL',
        'MYSQL',
        '8.0.19',
        'mysql.prod.example.com',
        '3306',
        'bytebase.com/database/mysql'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6005,
        101,
        101,
        5001,
        'Localhost Dev PostgreSQL',
        'POSTGRES',
        '13.0',
        '127.0.0.1',
        '5432',
        'bytebase.com/database/postgres'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6006,
        101,
        101,
        5001,
        'Demo PostgreSQL on Render',
        'POSTGRES',
        '13.0',
        'dpg-c8a7pcd0mal7gtod05p0',
        '5432',
        'postgres://postgre_demo_user:3QixNmRMGhklX6B1lmCZ3ZsHFPIE5EgG@dpg-c8a7pcd0mal7gtod05p0/postgre_demo'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6007,
        101,
        101,
        5001,
        'Localhost Dev TiDB',
        'TIDB',
        '5.7.25-TiDB-v5.2.1',
        '127.0.0.1',
        '4000',
        'bytebase.com/database/tidb'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6008,
        101,
        101,
        5001,
        'Localhost Dev ClickHouse',
        'CLICKHOUSE',
        '21.10.2.15',
        '127.0.0.1',
        '9000',
        'bytebase.com/database/clickhouse'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6009,
        101,
        101,
        5001,
        'Localhost Dev Snowflake',
        'SNOWFLAKE',
        '21.10.2.15',
        '127.0.0.1',
        '',
        'bytebase.com/database/snowflake'
    );

INSERT INTO
    instance (
        id,
        row_status,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        6010,
        'ARCHIVED',
        101,
        101,
        5004,
        'Retired Prod MySQL',
        'MYSQL',
        '5.7.25',
        'mysql.retired.example.com',
        '3306',
        'bytebase.com/database/mysql'
    );

ALTER SEQUENCE instance_id_seq RESTART WITH 6011;
