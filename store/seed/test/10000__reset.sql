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

-- Project 1 refers to DEFAULT project which is considered as part of schema
DELETE FROM
    project
WHERE
    id != 1;

DELETE FROM
    member;

-- Principal 1 refers to bytebase system account which is considered as part of schema
DELETE FROM
    principal
WHERE
    id != 1;

-- By default, for every table, we reserve id <=100 for internal user
UPDATE
    sqlite_sequence
SET
    seq = 100;