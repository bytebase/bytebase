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
        `when`
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
        'Waiting approval',
        'bb.task.approve',
        'PENDING',
        'MANUAL'
    );

-- Task for Pipeline 9002 schema update
-- Taks for stage 10002 "Sandbox A" to update database 7002 'fakedb1'
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
        `when`
    )
VALUES
    (
        11002,
        1001,
        1001,
        1,
        9002,
        10002,
        7002,
        'Update fakedb1',
        'bb.task.database.schema.update',
        'PENDING',
        'ON_SUCCESS'
    );

-- Taks for stage 10003 "Integration" to update database 7004 'fakedb2'
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
        `when`
    )
VALUES
    (
        11003,
        1001,
        1001,
        1,
        9002,
        10003,
        7004,
        'Update fakedb2',
        'bb.task.database.schema.update',
        'PENDING',
        'ON_SUCCESS'
    );

-- Taks for stage 10004 "Staging" to update database 7006 'fakedb3'
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
        `when`
    )
VALUES
    (
        11004,
        1001,
        1001,
        1,
        9002,
        10004,
        7006,
        'Update fakedb3',
        'bb.task.database.schema.update',
        'PENDING',
        'MANUAL'
    );

-- Taks for stage 10005 "Prod" to update database 7008 'fakedb4'
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
        `when`
    )
VALUES
    (
        11005,
        1001,
        1001,
        1,
        9002,
        10005,
        7008,
        'Update fakedb4',
        'bb.task.database.schema.update',
        'PENDING',
        'MANUAL'
    );