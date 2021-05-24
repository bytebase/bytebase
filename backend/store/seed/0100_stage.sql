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

-- Stage for Pipeline 9002 add column
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
        1003,
        1003,
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
        1003,
        1003,
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
        1003,
        1003,
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
        1003,
        1003,
        1,
        9002,
        5004,
        'Prod',
        'bb.stage.schema.update'
    );

-- Stage for Pipeline 9003 create table
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
        10006,
        1003,
        1003,
        1,
        9003,
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
        10007,
        1003,
        1003,
        1,
        9003,
        5002,
        'Integration',
        'bb.stage.schema.update'
    );