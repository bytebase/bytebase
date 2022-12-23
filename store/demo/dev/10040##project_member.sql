-- Project 3001 membership
INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4001,
        101,
        101,
        3001,
        'DEVELOPER',
        101
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4002,
        101,
        101,
        3001,
        'OWNER',
        102
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4003,
        101,
        101,
        3001,
        'DEVELOPER',
        103
    );

-- Project 3002 membership
INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4004,
        101,
        101,
        3002,
        'OWNER',
        101
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4005,
        101,
        101,
        3002,
        'DEVELOPER',
        102
    );

-- Project 3003 membership
INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4006,
        101,
        101,
        3003,
        'OWNER',
        101
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4007,
        101,
        101,
        3003,
        'DEVELOPER',
        103
    );

-- Project 3004 membership
INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4008,
        101,
        101,
        3004,
        'OWNER',
        101
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4009,
        101,
        101,
        3004,
        'OWNER',
        102
    );

INSERT INTO
    project_member (
        id,
        creator_id,
        updater_id,
        project_id,
        role,
        principal_id
    )
VALUES
    (
        4010,
        101,
        101,
        3004,
        'DEVELOPER',
        103
    );

ALTER SEQUENCE project_member_id_seq RESTART WITH 4011;
