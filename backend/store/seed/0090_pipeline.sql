-- A single stage, single step "hello world" pipeline
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `status`
    )
VALUES
    (
        9001,
        1001,
        1001,
        1,
        'Pipeline - Hello world',
        'OPEN'
    );

-- A multi stage, each containing a single step schema update pipeline
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        workspace_id,
        name,
        `status`
    )
VALUES
    (
        9002,
        1003,
        1003,
        1,
        'Pipeline - Add column ''location'' to table ''warehouse''',
        'OPEN'
    );