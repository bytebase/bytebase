-- Stage for Pipeline 9001 "Hello world"
INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10001,
        101,
        101,
        9001,
        5004,
        'Prod'
    );

-- Stage for Pipeline 9002 add column
INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10002,
        103,
        103,
        9002,
        5001,
        'Sandbox A'
    );

INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10003,
        103,
        103,
        9002,
        5002,
        'Integration'
    );

INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10004,
        103,
        103,
        9002,
        5003,
        'Staging'
    );

INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10005,
        103,
        103,
        9002,
        5004,
        'Prod'
    );

-- Stage for Pipeline 9003 create table
INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10006,
        103,
        103,
        9003,
        5001,
        'Sandbox A'
    );

INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10007,
        103,
        103,
        9003,
        5002,
        'Integration'
    );

-- Stage for Pipeline 9004 simulating webhook push event
INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        10008,
        1,
        1,
        9004,
        5003,
        'Staging'
    );