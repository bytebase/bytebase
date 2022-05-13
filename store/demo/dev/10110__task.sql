-- Task for Pipeline 9001 "Hello world", Stage 10001 "Prod" to update database 7014 "testdb_prod"
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
        type,
        status,
        payload
    )
VALUES
    (
        11001,
        101,
        101,
        9001,
        10001,
        6004,
        7014,
        'Welcome',
        'bb.task.general',
        'PENDING_APPROVAL',
        '{"statement":"SELECT ''Welcome Tech Lead, DBA, Developer'';"}'
    );

-- Task for Pipeline 9002
-- Task for stage 10002 "Dev" to update database 7003 'shop'
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11002,
        1,
        1624873710,
        1,
        1624873710,
        9002,
        10002,
        6001,
        7003,
        'Add initial schema',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10003 "Integration" to update database 7007 'shop'
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11003,
        1,
        1624873710,
        1,
        1624873710,
        9002,
        10003,
        6001,
        7007,
        'Add initial schema',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10004 "Staging" to update database 7011 'shop'
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11004,
        1,
        1624873710,
        1,
        1624873710,
        9002,
        10004,
        6003,
        7011,
        'Add initial schema',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10005 "Prod" to update database 7015 'proddb1'
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11005,
        1,
        1624873710,
        1,
        1624873710,
        9002,
        10005,
        6004,
        7015,
        'Add initial schema',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}}}'
    );

-- Task for Pipeline 9003 create table
-- Task for stage 10006 "Dev" to craete table in database 7002 'testdb_dev'
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
        type,
        status,
        payload
    )
VALUES
    (
        11006,
        103,
        103,
        9003,
        10006,
        6001,
        7002,
        'Update testdb_dev',
        'bb.task.database.schema.update',
        'FAILED',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_dev.tbl1 (name TEXT);"}'
    );

-- Task for stage 10003 "Integration" to update database 7006 'testdb_integration'
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
        type,
        status,
        payload
    )
VALUES
    (
        11007,
        103,
        103,
        9003,
        10007,
        6001,
        7006,
        'Update testdb_integration',
        'bb.task.database.schema.update',
        'PENDING',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_integration.tbl1 (name TEXT);"}'
    );

-- Task for stage 10008 simulating webhook push event
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11008,
        1,
        1624865387,
        1,
        1624865387,
        9004,
        10008,
        6001,
        7004,
        'Add initial schema',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8f578d53e821c46421d69fd0aabd29921190a6c0","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8f578d53e821c46421d69fd0aabd29921190a6c0","authorName":"tianzhou","added":"bytebase/dev/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10009 simulating webhook push event
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11009,
        1,
        1624866790,
        1,
        1624866790,
        9005,
        10009,
        6002,
        7008,
        'Add initial schema',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"0765950c28c50dd07217aa0a133e8239e64f0ea1","title":"Create user, post, comment table for integration environment","message":"Create user, post, comment table for integration environment","createdTs":1624866787,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/0765950c28c50dd07217aa0a133e8239e64f0ea1","authorName":"tianzhou","added":"bytebase/integration/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10010 simulating webhook push event
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11010,
        1,
        1624868407,
        1,
        1624868407,
        9006,
        10010,
        6003,
        7012,
        'Add initial schema',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"149f18b8d94805be4b92e256ef1dc9f4a27d5157","title":"Create user, post, comment table for staging environment","message":"Create user, post, comment table for staging environment","createdTs":1624868403,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/149f18b8d94805be4b92e256ef1dc9f4a27d5157","authorName":"tianzhou","added":"bytebase/bytebase/staging/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10011 simulating webhook push event
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11011,
        1,
        1624868680,
        1,
        1624868680,
        9007,
        10011,
        6004,
        7016,
        'Add initial schema',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"ec17d38d6581015feb49b341cc3da56cb0e354fb","title":"Create user, post, comment table for prod environment","message":"Create user, post, comment table for prod environment","createdTs":1624868676,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/ec17d38d6581015feb49b341cc3da56cb0e354fb","authorName":"tianzhou","added":"bytebase/prod/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task for stage 10012 simulating webhook push event
INSERT INTO
    task (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        pipeline_id,
        stage_id,
        instance_id,
        database_id,
        name,
        type,
        status,
        payload
    )
VALUES
    (
        11012,
        1,
        1624869944,
        1,
        1624869944,
        9008,
        10012,
        6001,
        7004,
        'Add created at column',
        'bb.task.database.schema.update',
        'FAILED',
        '{"migrationType":"MIGRATE","statement":"ALTER TABLE `user` ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE post ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE comment ADD COLUMN `created_at` DATETIME NOT NULL;\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8ff6482998059a861e1faa14658c65244577b54e","title":"Add created_at column to user,post,comment table for dev environment","message":"Add created_at column to user,post,comment table for dev environment","createdTs":1624869938,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8ff6482998059a861e1faa14658c65244577b54e","authorName":"tianzhou","added":"bytebase/dev/blog__202106280100__migrate__add_created_at_column.sql"}}}'
    );

-- Task for Pipeline 9009 multi-stage create table UI workflow
-- Task for stage 10013 "Dev" to create table in database 7002 'testdb_dev'
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
        type,
        status,
        payload
    )
VALUES
    (
        11013,
        103,
        103,
        9009,
        10013,
        6001,
        7002,
        'Update testdb_dev',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_dev.tbl2 (name TEXT);"}'
    );

-- Task for stage 10014 "Integration" to create table in database 7006 'testdb_integration'
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
        type,
        status,
        payload
    )
VALUES
    (
        11014,
        103,
        103,
        9009,
        10014,
        6002,
        7006,
        'Update testdb_integration',
        'bb.task.database.schema.update',
        'DONE',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_integration.tbl2 (name TEXT);"}'
    );


-- Task for stage 10015 "Integration" to create table in database 7010 'testdb_staging'
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
        type,
        status,
        payload
    )
VALUES
    (
        11015,
        103,
        103,
        9009,
        10015,
        6003,
        7010,
        'Update testdb_staging',
        'bb.task.database.schema.update',
        'FAILED',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_staging.tbl2 (name TEXT);"}'
    );

-- Task for stage 10016 "prod" to create table in database 7014 'testdb_prod'
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
        type,
        status,
        payload
    )
VALUES
    (
        11016,
        103,
        103,
        9009,
        10016,
        6004,
        7014,
        'Update testdb_prod',
        'bb.task.database.schema.update',
        'PENDING_APPROVAL',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_prod.tbl2 (name TEXT);"}'
    );

-- Tasks for task dependency, which is a gh-ost migration pipeline
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
        type,
        status,
        payload
    )
VALUES
    (
        11017,
        1,
        1,
        9010,
        10017,
        6001,
        7002,
        'Update testdb_dev gh-ost sync',
        'bb.task.database.schema.update.ghost.sync',
        'PENDING_APPROVAL',
        '{"statement":"ALTER TABLE tbl1 ENGINE=InnoDB;"}'
    );

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
        type,
        status,
        payload
    )
VALUES
    (
        11018,
        1,
        1,
        9010,
        10017,
        6001,
        7002,
        'Update testdb_dev gh-ost cutover',
        'bb.task.database.schema.update.ghost.cutover',
        'PENDING_APPROVAL',
        '{}'
    );

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
        type,
        status,
        payload
    )
VALUES
    (
        11019,
        1,
        1,
        9010,
        10017,
        6001,
        7002,
        'Update testdb_dev gh-ost drop original table',
        'bb.task.database.schema.update.ghost.drop-original-table',
        'PENDING_APPROVAL',
        '{"databaseName":"testdb_prod","tableName":"_tbl1_del"}'
    );

-- Tasks for PITR pipeline
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
        type,
        status,
        payload
    )
VALUES
    (
        11020,
        1,
        1,
        9011,
        10018,
        6001,
        7002,
        'Restore PITR database testdb_dev',
        'bb.task.database.pitr.restore',
        'PENDING_APPROVAL',
        '{"projectId":3001,"pointInTimeTs":1652429962}'
    ),
    (
        11021,
        1,
        1,
        9011,
        10018,
        6001,
        7002,
        'Swap PITR and the original database testdb_dev',
        'bb.task.database.pitr.cutover',
        'PENDING_APPROVAL',
        '{}'
    ),
    (
        11022,
        1,
        1,
        9011,
        10018,
        6001,
        7002,
        'Delete the original database testdb_dev',
        'bb.task.database.pitr.delete',
        'PENDING_APPROVAL',
        '{}'
    );

ALTER SEQUENCE task_id_seq RESTART WITH 11023;
