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
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_NEVER"}'
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
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_NEVER"}'
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
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_ALWAYS"}'
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
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_NEVER"}'
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
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_NEVER"}'
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
        5106,
        101,
        101,
        5003,
        'bb.policy.backup-plan',
        '{"schedule":"WEEKLY"}'
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
        5107,
        101,
        101,
        5004,
        'bb.policy.backup-plan',
        '{"schedule":"DAILY"}'
    );

-- Test upsert.
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
        5108,
        101,
        101,
        5004,
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_ALWAYS"}'
    )
    ON CONFLICT(environment_id, type) DO UPDATE SET
				payload = excluded.payload;

ALTER SEQUENCE policy_id_seq RESTART WITH 5109;
