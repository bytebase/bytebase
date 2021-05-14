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