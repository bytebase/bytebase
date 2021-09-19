INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`,
        approval_policy
    )
VALUES
    (
        5001,
        101,
        101,
        'Dev',
        0,
        'MANUAL_APPROVAL_NEVER'
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`,
        approval_policy
    )
VALUES
    (
        5002,
        101,
        101,
        'Integration',
        1,
        'MANUAL_APPROVAL_NEVER'
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`,
        approval_policy
    )
VALUES
    (
        5003,
        101,
        101,
        'Staging',
        2,
        'MANUAL_APPROVAL_ALWAYS'
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`,
        approval_policy
    )
VALUES
    (
        5004,
        101,
        101,
        'Prod',
        3,
        'MANUAL_APPROVAL_ALWAYS'
    );

INSERT INTO
    environment (
        id,
        row_status,
        creator_id,
        updater_id,
        name,
        `order`,
        approval_policy
    )
VALUES
    (
        5005,
        'ARCHIVED',
        101,
        101,
        'Archived Env 1',
        4,
        'MANUAL_APPROVAL_NEVER'
    );
