INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        5101,
        101,
        101,
        5001,
        'approval_policy',
        'MANUAL_APPROVAL_NEVER'
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        5102,
        101,
        101,
        5002,
        'approval_policy',
        'MANUAL_APPROVAL_NEVER'
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        5103,
        101,
        101,
        5003,
        'approval_policy',
        'MANUAL_APPROVAL_ALWAYS'
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        5104,
        101,
        101,
        5004,
        'approval_policy',
        'MANUAL_APPROVAL_NEVER'
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        5105,
        101,
        101,
        5005,
        'approval_policy',
        'MANUAL_APPROVAL_NEVER'
    );

-- Test upsert.
INSERT INTO
    policy (
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        101,
        101,
        5004,
        'approval_policy',
        'MANUAL_APPROVAL_ALWAYS'
    )
    ON CONFLICT(environment_id, type) DO UPDATE SET
				payload = excluded.payload;
