-- For testing, we reset data on each run
-- Do not reset bb.auth.secret so that we don't need to re-login after restart.
-- Do not reset bb.enterprise.license so that the dev license is preserved after restart.
DELETE FROM
    setting
WHERE
    name != 'bb.auth.secret' AND name != 'bb.enterprise.license';
-- We have to change the creator and updator to system robot, otherwise we will get foreigh key check error on
-- resetting principal table.
UPDATE setting SET creator_id = 1, updater_id = 1 WHERE name = 'bb.enterprise.license';

DELETE FROM
    anomaly;

DELETE FROM
    repository;

DELETE FROM
    vcs;

DELETE FROM
    bookmark;

DELETE FROM
    inbox;

DELETE FROM
    activity;

DELETE FROM
    sheet_organizer;

DELETE FROM
    sheet;

DELETE FROM
    issue_subscriber;

DELETE FROM
    issue;

DELETE FROM
    task_check_run;

DELETE FROM
    task_run;

DELETE FROM
    task_dag;

DELETE FROM
    task;

DELETE FROM
    stage;

DELETE FROM
    pipeline;

DELETE FROM
    data_source;

DELETE FROM
    vw;

DELETE FROM
    idx;

DELETE FROM
    col;

DELETE FROM
    tbl;

DELETE FROM
    backup;

DELETE FROM
    backup_setting;

-- Delete in this order following foreign constraints.
DELETE FROM
    db_label;

DELETE FROM
    label_value;

DELETE FROM
    label_key;

DELETE FROM
    db;

DELETE FROM
    instance_user;

DELETE FROM
    instance;

DELETE FROM
    policy;

DELETE FROM
    environment;

DELETE FROM
    project_webhook;

DELETE FROM
    project_member;

DELETE FROM
    deployment_config;

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
