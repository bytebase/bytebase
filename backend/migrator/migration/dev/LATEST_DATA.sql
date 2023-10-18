-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        101,
        1,
        1,
        'Test',
        0,
        'test'
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        102,
        1,
        1,
        'Prod',
        1,
        'prod'
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 103;

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        resource_type,
        resource_id,
        inherit_from_parent,
        type,
        payload
    )
VALUES
    (
        101,
        1,
        1,
        'ENVIRONMENT',
        101,
        TRUE,
        'bb.policy.rollout',
        '{"automatic": true}'
    );
