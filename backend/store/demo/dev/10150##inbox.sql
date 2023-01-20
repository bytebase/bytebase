-- Inbox for receiver 101
INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15001,
        101,
        14001,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15002,
        101,
        14003,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15003,
        101,
        14004,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15004,
        101,
        14005,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15005,
        101,
        14010,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15006,
        101,
        14012,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15007,
        101,
        14013,
        'UNREAD'
    );

-- Inbox for receiver 102
INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15008,
        102,
        14001,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15009,
        102,
        14002,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15010,
        102,
        14004,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15011,
        102,
        14005,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15012,
        102,
        14006,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15013,
        102,
        14013,
        'UNREAD'
    );

-- Inbox for receiver 103
INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15014,
        103,
        14001,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15015,
        103,
        14002,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15016,
        103,
        14003,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15017,
        103,
        14005,
        'READ'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15018,
        103,
        14006,
        'UNREAD'
    );

INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        status
    )
VALUES
    (
        15019,
        103,
        14013,
        'UNREAD'
    );

ALTER SEQUENCE inbox_id_seq RESTART WITH 15020;
