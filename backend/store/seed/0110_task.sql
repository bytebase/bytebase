-- Task for Pipeline 9001 "Hello world", Stage 10001 "Prod" to update database 7008 "fakedb4"
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
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
        1,
        9001,
        10001,
        7008,
        'Welcome',
        'bb.task.general',
        'PENDING_APPROVAL',
        '{"Sql":"SELECT ''Welcome Tech Lead, DBA, Developer'';"}'
    );

-- Task for Pipeline 9002 add column
-- Task for stage 10002 "Sandbox A" to update database 7002 'fakedb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
        database_id,
        name,
        `type`,
        `status`
    )
VALUES
    (
        11002,
        1003,
        1003,
        1,
        9002,
        10002,
        7002,
        'Update fakedb1',
        'bb.task.database.schema.update',
        'PENDING'
    );

-- Task for stage 10003 "Integration" to update database 7004 'fakedb2'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
        database_id,
        name,
        `type`,
        `status`
    )
VALUES
    (
        11003,
        1003,
        1003,
        1,
        9002,
        10003,
        7004,
        'Update fakedb2',
        'bb.task.database.schema.update',
        'PENDING'
    );

-- Task for stage 10004 "Staging" to update database 7006 'fakedb3'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
        database_id,
        name,
        `type`,
        `status`
    )
VALUES
    (
        11004,
        1003,
        1003,
        1,
        9002,
        10004,
        7006,
        'Update fakedb3',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL'
    );

-- Task for stage 10005 "Prod" to update database 7008 'fakedb4'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
        database_id,
        name,
        `type`,
        `status`
    )
VALUES
    (
        11005,
        1003,
        1003,
        1,
        9002,
        10005,
        7008,
        'Update fakedb4',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL'
    );

-- Task for Pipeline 9003 create table
-- Task for stage 10006 "Sandbox A" to craete table in database 7002 'fakedb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
        database_id,
        name,
        `type`,
        `status`
    )
VALUES
    (
        11006,
        1003,
        1003,
        1,
        9003,
        10006,
        7002,
        'Update fakedb1',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL'
    );

-- Task for stage 10003 "Integration" to update database 7004 'fakedb2'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        stage_id,
        database_id,
        name,
        `type`,
        `status`
    )
VALUES
    (
        11007,
        1003,
        1003,
        1,
        9003,
        10007,
        7004,
        'Update fakedb2',
        'bb.task.database.schema.update',
        'PENDING'
    );