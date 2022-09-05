-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        101,
        1,
        1,
        'Test',
        0
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        102,
        1,
        1,
        'Prod',
        1
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 103;

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
        101,
        1,
        1,
        101,
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
        102,
        1,
        1,
        102,
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
        103,
        1,
        1,
        102,
        'bb.policy.backup-plan',
        '{"schedule":"WEEKLY"}'
    );

ALTER SEQUENCE policy_id_seq RESTART WITH 104;

-- Create label keys for `bb.location` and `bb.tenant`.
INSERT INTO
    label_key (
        id,
        creator_id,
        updater_id,
        key
    )
VALUES
    (
        101,
        1,
        1,
        'bb.location'
    );

INSERT INTO
    label_key (
        id,
        creator_id,
        updater_id,
        key
    )
VALUES
    (
        102,
        1,
        1,
        'bb.tenant'
    );

ALTER SEQUENCE label_key_id_seq RESTART WITH 103;
