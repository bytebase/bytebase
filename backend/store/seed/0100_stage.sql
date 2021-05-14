-- Stage for Pipeline 9001 "Hello world"
INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        workspace_id,
        pipeline_id,
        environment_id,
        name,
        `type`
    )
VALUES
    (
        10001,
        1001,
        1001,
        1,
        9001,
        5004,
        'Prod',
        'bb.stage.schema.update'
    );