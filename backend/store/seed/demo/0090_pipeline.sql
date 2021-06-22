-- A single stage, single task "hello world" 
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        `status`
    )
VALUES
    (
        9001,
        101,
        101,
        'Pipeline - Hello world',
        'OPEN'
    );

-- A multi stage, each containing a single add column task
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        `status`
    )
VALUES
    (
        9002,
        103,
        103,
        'Pipeline - Add column ''location'' to table ''warehouse''',
        'OPEN'
    );

-- A two stage, each containing a single create table task, the 1st task has a failed task run
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        `status`
    )
VALUES
    (
        9003,
        103,
        103,
        'Pipeline - Create table ''tbl1''',
        'OPEN'
    );

-- A pipeline simulating webhook push event
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        `status`
    )
VALUES
    (
        9004,
        1,
        1,
        'Pipeline - Create todo table to staging db1',
        'OPEN'
    );