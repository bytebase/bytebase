-- Project 5001 membership
INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        `role`,
        principal_id
    )
VALUES
    (
        4001,
        1001,
        1001,
        1,
        3001,
        'OWNER',
        1002
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        `role`,
        principal_id
    )
VALUES
    (
        4002,
        1001,
        1001,
        1,
        3001,
        'DEVELOPER',
        1001
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        `role`,
        principal_id
    )
VALUES
    (
        4003,
        1001,
        1001,
        1,
        3001,
        'DEVELOPER',
        1003
    );

-- Project 5002 membership
INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        `role`,
        principal_id
    )
VALUES
    (
        4004,
        1001,
        1001,
        1,
        3002,
        'OWNER',
        1001
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        `role`,
        principal_id
    )
VALUES
    (
        4005,
        1001,
        1001,
        1,
        3002,
        'DEVELOPER',
        1003
    );