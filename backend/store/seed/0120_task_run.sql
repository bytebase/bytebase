-- Failed task run for task 11006 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        updater_id,
        workspace_id,
        task_id,
        name,
        `status`,
        `type`
    )
VALUES
    (
        12001,
        1003,
        1003,
        1,
        11006,
        'UPDATE fakedb1 task run',
        'FAILED',
        'bb.task.database.schema.update'
    );