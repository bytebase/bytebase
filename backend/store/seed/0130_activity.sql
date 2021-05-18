-- Activity for issue 012001
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
        13001,
        1,
        1,
        1,
        12001,
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
        13002,
        1001,
        1001,
        1,
        12001,
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
        13003,
        1002,
        1002,
        1,
        12001,
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
        13004,
        1003,
        1003,
        1,
        12001,
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
        13005,
        1,
        1,
        1,
        12002,
        'bb.issue.create'
    );