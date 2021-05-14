INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `order`
    )
VALUES
    (
        5001,
        1001,
        1001,
        1,
        'Sandbox A',
        0
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `order`
    )
VALUES
    (
        5002,
        1001,
        1001,
        1,
        'Integration',
        1
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `order`
    )
VALUES
    (
        5003,
        1001,
        1001,
        1,
        'Staging',
        2
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `order`
    )
VALUES
    (
        5004,
        1001,
        1001,
        1,
        'Prod',
        3
    );

INSERT INTO
    environment (
        id,
        row_status,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `order`
    )
VALUES
    (
        3005,
        'ARCHIVED',
        1001,
        1001,
        1,
        'Archived Env 1',
        4
    );