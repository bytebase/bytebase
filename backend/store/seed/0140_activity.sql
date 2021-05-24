-- Activity for issue 013001
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        workspace_id,
        container_id,
        `type`
    )
VALUES
    (
        14001,
        1,
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
        workspace_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        14002,
        1001,
        1001,
        1,
        13001,
        'bb.issue.comment.create',
        'Welcome!'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        workspace_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        14003,
        1002,
        1002,
        1,
        13001,
        'bb.issue.comment.create',
        'Let''s rock!'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        workspace_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        14004,
        1003,
        1003,
        1,
        13001,
        'bb.issue.comment.create',
        'Go fish!'
    );

-- Activity for issue 012002
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        workspace_id,
        container_id,
        `type`
    )
VALUES
    (
        14005,
        1,
        1,
        1,
        13002,
        'bb.issue.create'
    );