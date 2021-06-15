-- Task for Pipeline 9001 "Hello world", Stage 10001 "Prod" to update database 7008 "testdb4"
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11001,
        1001,
        1001,
        9001,
        10001,
        6004,
        7008,
        'Welcome',
        'bb.task.general',
        'PENDING_APPROVAL',
        '{"statement":"SELECT ''Welcome Tech Lead, DBA, Developer'';"}'
    );

-- Task for Pipeline 9002 add column
-- Task for stage 10002 "Sandbox A" to update database 7002 'testdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11002,
        1003,
        1003,
        9002,
        10002,
        6001,
        7002,
        'Update testdb1',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"ALTER TABLE testdb1.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE testdb1.warehouse DROP COLUMN location;"}'
    );

-- Task for stage 10003 "Integration" to update database 7003 'testdb2'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11003,
        1003,
        1003,
        9002,
        10003,
        6001,
        7003,
        'Update testdb2',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"ALTER TABLE testdb2.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE testdb2.warehouse DROP COLUMN location;"}'
    );

-- Task for stage 10004 "Staging" to update database 7006 'testdb3'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11004,
        1003,
        1003,
        9002,
        10004,
        6003,
        7006,
        'Update testdb3',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"statement":"ALTER TABLE testdb3.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE testdb3.warehouse DROP COLUMN location;"}'
    );

-- Task for stage 10005 "Prod" to update database 7008 'testdb4'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11005,
        1003,
        1003,
        9002,
        10005,
        6004,
        7008,
        'Update testdb4',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"statement":"ALTER TABLE testdb4.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE testdb4.warehouse DROP COLUMN location;"}'
    );

-- Task for Pipeline 9003 create table
-- Task for stage 10006 "Sandbox A" to craete table in database 7002 'testdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11006,
        1003,
        1003,
        9003,
        10006,
        6001,
        7002,
        'Update testdb1',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"statement":"CREATE TABLE testdb1.tbl1 (name TEXT);", "rollbackStatement":"DROP TABLE testdb1.tbl1;"}'
    );

-- Task for stage 10003 "Integration" to update database 7003 'testdb2'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11007,
        1003,
        1003,
        9003,
        10007,
        6001,
        7003,
        'Update testdb2',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"CREATE TABLE testdb2.tbl1 (name TEXT);", "rollbackStatement":"DROP TABLE testdb2.tbl1;"}'
    );