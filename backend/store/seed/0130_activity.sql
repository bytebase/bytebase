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
        'Glad to be here!'
    );