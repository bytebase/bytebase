-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (
        101,
        1,
        1,
        'Test',
        0
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (
        102,
        1,
        1,
        'Prod',
        1
    );

-- Create 1 "test" and 1 "prod" instance (including * database and admin data source)
-- "test" instance
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        101,
        1,
        1,
        101,
        'On-premises Test MySQL',
        'MYSQL',
        'mysql.test.example.com',
        '3306',
        'bytebase.com'
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        character_set,
        `collation`,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        1,
        1,
        101,
        1,
        '*',
        'utf8mb4',
        'utf8mb4_0900_ai_ci',
        'OK',
        0
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
        101,
        1,
        1,
        101,
        101,
        'Admin data source',
        'ADMIN',
        'admin',
        ''
    );

-- "prod" instance
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        102,
        1,
        1,
        102,
        'On-premises Prod MySQL',
        'MYSQL',
        'mysql.prod.example.com',
        '3306',
        'bytebase.com'
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        character_set,
        `collation`,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        102,
        1,
        1,
        102,
        1,
        '*',
        'utf8mb4',
        'utf8mb4_0900_ai_ci',
        'OK',
        0
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
        102,
        1,
        1,
        102,
        102,
        'Admin data source',
        'ADMIN',
        'admin',
        ''
    );