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
        5101,
        101,
        101,
        5004,
        'approval_policy',
        'MANUAL_APPROVAL_ALWAYS'
    )
    ON CONFLICT(environment_id, type) DO UPDATE SET
				payload = excluded.payload;
