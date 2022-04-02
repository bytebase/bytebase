INSERT INTO
    member (
        id,
        creator_id,
        updater_id,
        status,
        role,
        principal_id
    )
VALUES
    (2001, 101, 101, 'ACTIVE', 'OWNER', 101);

INSERT INTO
    member (
        id,
        creator_id,
        updater_id,
        status,
        role,
        principal_id
    )
VALUES
    (2002, 101, 101, 'ACTIVE', 'DBA', 102);

INSERT INTO
    member (
        id,
        creator_id,
        updater_id,
        status,
        role,
        principal_id
    )
VALUES
    (2003, 101, 101, 'ACTIVE', 'DEVELOPER', 103);

INSERT INTO
    member (
        id,
        creator_id,
        updater_id,
        status,
        role,
        principal_id
    )
VALUES
    (2004, 101, 101, 'ACTIVE', 'DEVELOPER', 104);

ALTER SEQUENCE member_id_seq RESTART WITH 2005;
