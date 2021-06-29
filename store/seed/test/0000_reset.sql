-- For testing, we reset data on each run
DELETE FROM
    repo;

DELETE FROM
    vcs;

DELETE FROM
    bookmark;

DELETE FROM
    activity;

DELETE FROM
    issue;

DELETE FROM
    task_run;

DELETE FROM
    task;

DELETE FROM
    stage;

DELETE FROM
    pipeline;

DELETE FROM
    data_source;

DELETE FROM
    tbl;

DELETE FROM
    db;

DELETE FROM
    instance;

DELETE FROM
    environment;

DELETE FROM
    project_member;

DELETE FROM
    project
WHERE
    id != 1;

DELETE FROM
    member;

DELETE FROM
    principal
WHERE
    id != 1;

UPDATE
    sqlite_sequence
SET
    seq = 100;