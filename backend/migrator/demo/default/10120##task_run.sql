-- Task run for task 11002
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12001,
        1,
        1624873710,
        1,
        1624873710,
        11002,
        0,
        'Add initial schema 1624873710',
        'DONE',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''shop''","migrationId":1,"version":"202106280000"}'
    );

-- Task run for task 11003
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12002,
        1,
        1624883710,
        1,
        1624883710,
        11003,
        0,
        'Add initial schema 1624883710',
        'DONE',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''shop''","migrationId":1,"version":"202106280000"}'
    );

-- Failed task run for task 11006 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        updater_id,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12003,
        101,
        101,
        11006,
        0,
        'Update testdb_dev task run',
        'FAILED',
        103,
        '{"detail":"table \"tbl1\" already exists"}'
    );

-- Task run for task 11008
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12004,
        1,
        1624865387,
        1,
        1624865387,
        11008,
        0,
        'Add initial schema 1624865387',
        'DONE',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}'
    );

-- Task run for task 11009
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12005,
        1,
        1624866790,
        1,
        1624866790,
        11009,
        0,
        'Add initial schema 1624866790',
        'DONE',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}'
    );

-- Task run for task 11010
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12006,
        1,
        1624868407,
        1,
        1624868407,
        11010,
        0,
        'Add initial schema 1624868407',
        'DONE',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}'
    );

-- Task run for task 11011
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12007,
        1,
        1624868680,
        1,
        1624868680,
        11011,
        0,
        'Add initial schema 1624868680',
        'DONE',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}'
    );

-- Failed task run for task 11012 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12008,
        1,
        1624869944,
        1,
        1624869944,
        11012,
        0,
        'Add created at column 1624869944',
        'FAILED',
        201,
        '{"detail":"database ''blog'' has already applied version 202106280100"}'
    );

-- Successful task run for task 11013 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12009,
        102,
        1624879944,
        102,
        1624879944,
        11013,
        0,
        'Update testdb_dev task run',
        'DONE',
        0,
        '{"detail":"Applied migration version 20210830011437.11013 to database \"testdb_dev\"","migrationId":1,"version":"20210830011437.11013"}'
    );

-- Successful task run for task 11014 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        result
    )
VALUES
    (
        12010,
        102,
        1624879944,
        102,
        1624879944,
        11014,
        0,
        'Update testdb_integration task run',
        'DONE',
        '{"detail":"Applied migration version 20210830011437.11014 to database \"testdb_integration\"","migrationId":1,"version":"20210830011437.11014"}'
    );

-- Failed task run for task 11015 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        attempt,
        name,
        status,
        code,
        result
    )
VALUES
    (
        12011,
        102,
        1624879944,
        102,
        1624879944,
        11015,
        0,
        'Update testdb_staging task run',
        'FAILED',
        103,
        '{"detail":"table \"tbl2\" already exists"}'
    );

ALTER SEQUENCE task_run_id_seq RESTART WITH 12012;
