-- A single stage, single task "hello world"
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        project_id,
        name
    )
VALUES
    (
        9001,
        101,
        101,
        3001,
        'Pipeline - Hello world'
    );

-- A multiple stage Pipeline for simulating webhook push event to create table for shop project database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9002,
        1,
        1624873710,
        1,
        1624873710,
        3002,
        'Pipeline - Create product table'
    );

-- A two stage, each containing a single create table task, the 1st task has a failed task run
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        project_id,
        name
    )
VALUES
    (
        9003,
        103,
        103,
        3001,
        'Pipeline - Create table ''tbl1'''
    );

-- Pipeline for simulating webhook push event to create table for blog project dev database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9004,
        1,
        1624865387,
        1,
        1624865387,
        3003,
        'Pipeline - Create user, post, comment table for dev environment'
    );

-- Pipeline for simulating webhook push event to create table for blog project integration database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9005,
        1,
        1624866790,
        1,
        1624866790,
        3003,
        'Pipeline - Create user, post, comment table for integration environment'
    );

-- Pipeline for simulating webhook push event to create table for blog project staging database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9006,
        1,
        1624868407,
        1,
        1624868407,
        3003,
        'Pipeline - Create user, post, comment table for staging environment'
    );

-- Pipeline for simulating webhook push event to create table for blog project prod database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9007,
        1,
        1624868680,
        1,
        1624868680,
        3003,
        'Pipeline - Create user, post, comment table for prod environment'
    );

-- Pipeline for simulating webhook push event to alter table for blog project dev database.
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9008,
        1,
        1624869944,
        1,
        1624869944,
        3003,
        'Pipeline - Add created_at column to user,post,comment table for dev environment'
    );

-- Pipeline for multi-stage create table UI workflow
INSERT INTO
    pipeline (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        project_id,
        name
    )
VALUES
    (
        9009,
        103,
        1624879944,
        103,
        1624879944,
        3001,
        'Pipeline - Create a new table tbl2'
    );

ALTER SEQUENCE pipeline_id_seq RESTART WITH 9010;
