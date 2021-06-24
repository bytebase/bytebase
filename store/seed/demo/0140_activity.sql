-- Activity for issue 13001
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`
    )
VALUES
    (
        14001,
        1,
        1,
        13001,
        'bb.issue.create'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        14002,
        101,
        101,
        13001,
        'bb.issue.comment.create',
        'Welcome!'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        14003,
        102,
        102,
        13001,
        'bb.issue.comment.create',
        'Let''s rock!'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        14004,
        103,
        103,
        13001,
        'bb.issue.comment.create',
        'Go fish!'
    );

-- Activity for issue 13002
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`
    )
VALUES
    (
        14005,
        103,
        103,
        13002,
        'bb.issue.create'
    );

-- Activity for issue 13003
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`
    )
VALUES
    (
        14006,
        103,
        103,
        13003,
        'bb.issue.create'
    );