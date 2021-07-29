-- Inbox for receiver 101
INSERT INTO
    inbox (
        id,
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (
        15001,
        101,
        14001,
        'READ',
        'INFO'
    );

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (101, 14003, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (101, 14004, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (101, 14005, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (101, 14010, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (101, 14012, 'UNREAD', 'ERROR');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (101, 14013, 'UNREAD', 'INFO');

-- Inbox for receiver 102
INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (102, 14001, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (102, 14002, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (102, 14004, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (102, 14005, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (102, 14006, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (102, 14013, 'UNREAD', 'INFO');

-- Inbox for receiver 103
INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (103, 14001, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (103, 14002, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (103, 14003, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (103, 14005, 'READ', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (103, 14006, 'UNREAD', 'INFO');

INSERT INTO
    inbox (
        receiver_id,
        activity_id,
        `status`,
        `level`
    )
VALUES
    (103, 14013, 'UNREAD', 'INFO');