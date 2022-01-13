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
        receiver_id,
        activity_id,
        status
    )
VALUES
    (101, 14003, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (101, 14004, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (101, 14005, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (101, 14010, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (101, 14012, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (101, 14013, 'UNREAD');

-- Inbox for receiver 102
INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (102, 14001, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (102, 14002, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (102, 14004, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (102, 14005, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (102, 14006, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (102, 14013, 'UNREAD');

-- Inbox for receiver 103
INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (103, 14001, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (103, 14002, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (103, 14003, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (103, 14005, 'READ');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (103, 14006, 'UNREAD');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        status
    )
VALUES
    (103, 14013, 'UNREAD');
