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
        6001,
        1001,
        1001,
        1,
        5001,
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
        6002,
        1001,
        1001,
        1,
        5001,
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
        6003,
        1001,
        1001,
        1,
        5001,
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
        6005,
        1001,
        1001,
        1,
        5002,
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
        6006,
        1001,
        1001,
        1,
        5002,
        'DEVELOPER',
        1003
    );