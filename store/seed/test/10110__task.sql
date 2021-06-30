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
        `type`,
        `status`,
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"14","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repoFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"da90a2510eccd051ad14e4b89ca904d733169a39","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/da90a2510eccd051ad14e4b89ca904d733169a39","authorName":"tianzhou","added":"bytebase/v1__shop__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"14","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repoFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"da90a2510eccd051ad14e4b89ca904d733169a39","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/da90a2510eccd051ad14e4b89ca904d733169a39","authorName":"tianzhou","added":"bytebase/v1__shop__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"14","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repoFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"da90a2510eccd051ad14e4b89ca904d733169a39","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/da90a2510eccd051ad14e4b89ca904d733169a39","authorName":"tianzhou","added":"bytebase/v1__shop__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"14","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repoFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"da90a2510eccd051ad14e4b89ca904d733169a39","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/da90a2510eccd051ad14e4b89ca904d733169a39","authorName":"tianzhou","added":"bytebase/v1__shop__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE testdb_dev.tbl1 (name TEXT);", "rollbackStatement":"DROP TABLE testdb_dev.tbl1;"}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE testdb_integration.tbl1 (name TEXT);", "rollbackStatement":"DROP TABLE testdb_integration.tbl1;"}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","authorName":"tianzhou","added":"bytebase/dev/202106280000__blog__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"70d06befa6d78abed1b78b7de1cd0e7ce3365719","title":"Create user, post, comment table for integration environment","message":"Create user, post, comment table for integration environment","createdTs":1624866787,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/70d06befa6d78abed1b78b7de1cd0e7ce3365719","authorName":"tianzhou","added":"bytebase/integration/202106280000__blog__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"2dac551a44ff26b08d20bd213b856a993da50ecb","title":"Create user, post, comment table for staging environment","message":"Create user, post, comment table for staging environment","createdTs":1624868403,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/2dac551a44ff26b08d20bd213b856a993da50ecb","authorName":"tianzhou","added":"bytebase/staging/202106280000__blog__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"e3ff00e2432cfb2da422222ab4f055dd43c4d441","title":"Create user, post, comment table for prod environment","message":"Create user, post, comment table for prod environment","createdTs":1624868676,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/e3ff00e2432cfb2da422222ab4f055dd43c4d441","authorName":"tianzhou","added":"bytebase/prod/202106280000__blog__baseline__add_initial_schema.sql"}}}'
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
        `type`,
        `status`,
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
        '{"statement":"ALTER TABLE `user` ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE post ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE comment ADD COLUMN `created_at` DATETIME NOT NULL;\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"171ceaf7659ceb8e495aa3ef356ec686656f9dc0","title":"Add created_at column to user,post,comment table for dev environment","message":"Add created_at column to user,post,comment table for dev environment","createdTs":1624869938,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/171ceaf7659ceb8e495aa3ef356ec686656f9dc0","authorName":"tianzhou","added":"bytebase/dev/202106280100__blog__add_created_at_column.sql"}}}'
    );