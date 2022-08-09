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
        104,
        1,
        1,
        101,
        'bb.policy.assignee-choice',
        '{"value":"WORKSPACE_OWNER_OR_DBA"}'
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
        105,
        1,
        1,
        102,
        'bb.policy.assignee-choice',
        '{"value":"WORKSPACE_OWNER_OR_DBA"}'
    );

ALTER SEQUENCE policy_id_seq RESTART WITH 106;