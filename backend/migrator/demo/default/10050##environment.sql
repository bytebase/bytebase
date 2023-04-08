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
        5001,
        101,
        101,
        'Dev',
        0,
        'dev'
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
        5002,
        101,
        101,
        'Integration',
        1,
        'integration'
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
        5003,
        101,
        101,
        'Staging',
        2,
        'staging'
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
        5004,
        101,
        101,
        'Prod',
        3,
        'prod'
    );

INSERT INTO
    environment (
        id,
        row_status,
        creator_id,
        updater_id,
        name,
        "order",
        resource_id
    )
VALUES
    (
        5005,
        'ARCHIVED',
        101,
        101,
        'Archived Env 1',
        4,
        'archived'
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 5006;
