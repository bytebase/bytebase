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
        1001,
        1001,
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
        1003,
        1003,
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
        1003,
        1003,
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
        1003,
        1003,
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
        1003,
        1003,
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
        1003,
        1003,
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
        1003,
        1003,
        9003,
        5002,
        'Integration'
    );