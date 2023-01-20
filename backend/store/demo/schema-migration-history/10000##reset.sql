-- delete all table records
DELETE FROM
    setting;

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

DELETE FROM
    project;

DELETE FROM
    member;

DELETE FROM
    principal;
