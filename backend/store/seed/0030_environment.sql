INSERT INTO
    environment (
        id,
        workspace_id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (3001, 1, 1001, 1001, 'Sandbox A', 0);

INSERT INTO
    environment (
        id,
        workspace_id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (3002, 1, 1001, 1001, 'Integration', 1);

INSERT INTO
    environment (
        id,
        workspace_id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (3003, 1, 1001, 1001, 'Staging', 2);

INSERT INTO
    environment (
        id,
        workspace_id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (3004, 1, 1001, 1001, 'Prod', 3);

INSERT INTO
    environment (
        id,
        row_status,
        workspace_id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (
        3005,
        'ARCHIVED',
        1,
        1001,
        1001,
        'Archived Env 1',
        4
    );