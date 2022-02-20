-- A single stage, single task "hello world"
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        status
    )
VALUES
    (
        9001,
        101,
        101,
        'Pipeline - Hello world',
        'OPEN'
    );

-- A multiple stage Pipeline for simulating webhook push event to create table for shop project database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9002,
        1,
        1624873710,
        1,
        1624873710,
        'Pipeline - Create product table',
        'OPEN'
    );

-- A two stage, each containing a single create table task, the 1st task has a failed task run
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        status
    )
VALUES
    (
        9003,
        103,
        103,
        'Pipeline - Create table ''tbl1''',
        'CANCELED'
    );

-- Pipeline for simulating webhook push event to create table for blog project dev database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9004,
        1,
        1624865387,
        1,
        1624865387,
        'Pipeline - Create user, post, comment table for dev environment',
        'DONE'
    );

-- Pipeline for simulating webhook push event to create table for blog project integration database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9005,
        1,
        1624866790,
        1,
        1624866790,
        'Pipeline - Create user, post, comment table for integration environment',
        'DONE'
    );

-- Pipeline for simulating webhook push event to create table for blog project staging database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9006,
        1,
        1624868407,
        1,
        1624868407,
        'Pipeline - Create user, post, comment table for staging environment',
        'DONE'
    );

-- Pipeline for simulating webhook push event to create table for blog project prod database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9007,
        1,
        1624868680,
        1,
        1624868680,
        'Pipeline - Create user, post, comment table for prod environment',
        'DONE'
    );

-- Pipeline for simulating webhook push event to alter table for blog project dev database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9008,
        1,
        1624869944,
        1,
        1624869944,
        'Pipeline - Add created_at column to user,post,comment table for dev environment',
        'OPEN'
    );

-- Pipeline for multi-stage create table UI workflow
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        name,
        status
    )
VALUES
    (
        9009,
        103,
        1624879944,
        103,
        1624879944,
        'Pipeline - Create a new table tbl2',
        'OPEN'
    );

ALTER SEQUENCE pipeline_id_seq RESTART WITH 9010;
