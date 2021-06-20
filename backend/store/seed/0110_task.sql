-- Task for Pipeline 9001 "Hello world", Stage 10001 "Prod" to update database 7008 "proddb1"
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11001,
        1001,
        1001,
        9001,
        10001,
        6004,
        7014,
        'Welcome',
        'bb.task.general',
        'PENDING_APPROVAL',
        '{"statement":"SELECT ''Welcome Tech Lead, DBA, Developer'';"}'
    );

-- Task for Pipeline 9002 add column
-- Task for stage 10002 "Sandbox A" to update database 7002 'sandboxdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11002,
        1003,
        1003,
        9002,
        10002,
        6001,
        7002,
        'Update sandboxdb1',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"ALTER TABLE sandboxdb1.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE sandboxdb1.warehouse DROP COLUMN location;"}'
    );

-- Task for stage 10003 "Integration" to update database 7006 'integrationdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11003,
        1003,
        1003,
        9002,
        10003,
        6001,
        7006,
        'Update integrationdb1',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"ALTER TABLE integrationdb1.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE integrationdb1.warehouse DROP COLUMN location;"}'
    );

-- Task for stage 10004 "Staging" to update database 7010 'stagingdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11004,
        1003,
        1003,
        9002,
        10004,
        6003,
        7010,
        'Update stagingdb1',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"statement":"ALTER TABLE stagingdb1.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE stagingdb1.warehouse DROP COLUMN location;"}'
    );

-- Task for stage 10005 "Prod" to update database 7014 'proddb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11005,
        1003,
        1003,
        9002,
        10005,
        6004,
        7014,
        'Update proddb1',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"statement":"ALTER TABLE proddb1.warehouse ADD COLUMN location VARCHAR(255);", "rollbackStatement":"ALTER TABLE proddb1.warehouse DROP COLUMN location;"}'
    );

-- Task for Pipeline 9003 create table
-- Task for stage 10006 "Sandbox A" to craete table in database 7002 'sandboxdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11006,
        1003,
        1003,
        9003,
        10006,
        6001,
        7002,
        'Update sandboxdb1',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"statement":"CREATE TABLE sandboxdb1.tbl1 (name TEXT);", "rollbackStatement":"DROP TABLE sandboxdb1.tbl1;"}'
    );

-- Task for stage 10003 "Integration" to update database 7006 'integrationdb1'
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11007,
        1003,
        1003,
        9003,
        10007,
        6001,
        7006,
        'Update integrationdb1',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"CREATE TABLE integrationdb1.tbl1 (name TEXT);", "rollbackStatement":"DROP TABLE integrationdb1.tbl1;"}'
    );

-- Task for stage 10008 "Staging" to update database 7010 'stagingdb1' to simulate webhook push event
INSERT INTO
    task (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        `type`,
        `status`,
        payload
    )
VALUES
    (
        11008,
        1,
        1,
        9004,
        10008,
        6003,
        7010,
        'create todo table',
        'bb.task.database.schema.update',
        'PENDING',
        '{"statement":"CREATE TABLE todo (\n  id INTEGER PRIMARY KEY AUTO_INCREMENT,\n  title TEXT\n)\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","ref":"refs/heads/master","repoId":"7","repoUrl":"http://gitlab.bytebase.com/bytebase-test/project1","repoFullPath":"bytebase-test/project1","authorName":"tianzhou","fileCommit":{"id":"e3b3f33a455a7e3cd999b44fa9cb8ddb633e1d29","title":"Create todo table to staging db1","message":"Create todo table to staging db1","createdTs":1624006500,"url":"http://gitlab.bytebase.com/bytebase-test/project1/-/commit/e3b3f33a455a7e3cd999b44fa9cb8ddb633e1d29","authorName":"tianzhou","added":"bytebase/20210618164921_stagingdb1_create_todo_table.sql"}}}'
    );