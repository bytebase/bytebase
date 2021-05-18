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

-- Stage for Pipeline 9002 schema update
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
        10002,
        1001,
        1001,
        1,
        9002,
        5001,
        'Sandbox A',
        'bb.stage.schema.update'
    );

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
        10003,
        1001,
        1001,
        1,
        9002,
        5002,
        'Integration',
        'bb.stage.schema.update'
    );

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
        10004,
        1001,
        1001,
        1,
        9002,
        5003,
        'Staging',
        'bb.stage.schema.update'
    );

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
        10005,
        1001,
        1001,
        1,
        9002,
        5004,
        'Prod',
        'bb.stage.schema.update'
    );