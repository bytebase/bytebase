-- End user id starts at 1001, we reserve the range between 1 ~ 1000 for internal use.
-- Setting the id explicitly changes the next id value to be +1
INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1001,
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Demo Owner',
        'demo@example.com',
        ''
    );

INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1002,
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Jerry DBA',
        'jerry@example.com',
        ''
    );

INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1003,
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Tom Dev',
        'tom@example.com',
        ''
    );

INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1004,
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Alice DBA',
        'alice@example.com',
        ''
    );

INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1005,
        1,
        1,
        'ACTIVE',
        'END_USER',
        'Jane Dev',
        'jane@example.com',
        ''
    );

INSERT INTO
    principal (
        id,
        creator_id,
        updater_id,
        `status`,
        `type`,
        name,
        email,
        password_hash
    )
VALUES
    (
        1006,
        1,
        1,
        'INVITED',
        'END_USER',
        'Bob Invited',
        'bob@example.com',
        ''
    );