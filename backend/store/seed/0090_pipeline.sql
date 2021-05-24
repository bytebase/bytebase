-- A single stage, single task "hello world" 
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

-- A multi stage, each containing a single add column task
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

-- A two stage, each containing a single create table task, the 1st task has a failed task run
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
        9003,
        1003,
        1003,
        1,
        'Pipeline - Create table ''tbl1''',
        'OPEN'
    );