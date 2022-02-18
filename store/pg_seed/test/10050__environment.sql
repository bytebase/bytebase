INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        5001,
        101,
        101,
        'Dev',
        0
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        5002,
        101,
        101,
        'Integration',
        1
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        5003,
        101,
        101,
        'Staging',
        2
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        5004,
        101,
        101,
        'Prod',
        3
    );

INSERT INTO
    environment (
        id,
        row_status,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        5005,
        'ARCHIVED',
        101,
        101,
        'Archived Env 1',
        4
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 5006;
