-- Failed task run for task 11006 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        updater_id,
        task_id,
        name,
        `status`,
        `type`,
        error
    )
VALUES
    (
        12001,
        1001,
        1001,
        11006,
        'UPDATE testdb1 task run',
        'FAILED',
        'bb.task.database.schema.update',
        'fake execution error'
    );