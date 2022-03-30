-- Task run for task 11002
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12001,
        1,
        1624873710,
        1,
        1624873710,
        11002,
        'Add initial schema 1624873710',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''shop''","migrationId":1,"version":"202106280000"}',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11003
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12002,
        1,
        1624883710,
        1,
        1624883710,
        11003,
        'Add initial schema 1624883710',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''shop''","migrationId":1,"version":"202106280000"}',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}}}'
    );

-- Failed task run for task 11006 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        updater_id,
        task_id,
        name,
        status,
        type,
        code,
        result
    )
VALUES
    (
        12003,
        101,
        101,
        11006,
        'Update testdb_dev task run',
        'FAILED',
        'bb.task.database.schema.update',
        103,
        '{"detail":"table \"tbl1\" already exists"}'
    );

-- Task run for task 11008
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12004,
        1,
        1624865387,
        1,
        1624865387,
        11008,
        'Add initial schema 1624865387',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8f578d53e821c46421d69fd0aabd29921190a6c0","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8f578d53e821c46421d69fd0aabd29921190a6c0","authorName":"tianzhou","added":"bytebase/dev/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11009
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12005,
        1,
        1624866790,
        1,
        1624866790,
        11009,
        'Add initial schema 1624866790',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8f578d53e821c46421d69fd0aabd29921190a6c0","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8f578d53e821c46421d69fd0aabd29921190a6c0","authorName":"tianzhou","added":"bytebase/dev/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11010
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12006,
        1,
        1624868407,
        1,
        1624868407,
        11010,
        'Add initial schema 1624868407',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8f578d53e821c46421d69fd0aabd29921190a6c0","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8f578d53e821c46421d69fd0aabd29921190a6c0","authorName":"tianzhou","added":"bytebase/dev/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11011
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12007,
        1,
        1624868680,
        1,
        1624868680,
        11011,
        'Add initial schema 1624868680',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Established baseline version 202106280000 for database ''blog''","migrationId":1,"version":"202106280000"}',
        '{"migrationType":"BASELINE","statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8f578d53e821c46421d69fd0aabd29921190a6c0","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8f578d53e821c46421d69fd0aabd29921190a6c0","authorName":"tianzhou","added":"bytebase/dev/blog__202106280000__baseline__add_initial_schema.sql"}}}'
    );

-- Failed task run for task 11012 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12008,
        1,
        1624869944,
        1,
        1624869944,
        11012,
        'Add created at column 1624869944',
        'FAILED',
        'bb.task.database.schema.update',
        201,
        '{"detail":"database ''blog'' has already applied version 202106280100"}',
        '{"migrationType":"MIGRATE","statement":"ALTER TABLE `user` ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE post ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE comment ADD COLUMN `created_at` DATETIME NOT NULL;\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8ff6482998059a861e1faa14658c65244577b54e","title":"Add created_at column to user,post,comment table for dev environment","message":"Add created_at column to user,post,comment table for dev environment","createdTs":1624869938,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8ff6482998059a861e1faa14658c65244577b54e","authorName":"tianzhou","added":"bytebase/dev/blog__202106280100__migrate__add_created_at_column.sql"}}}'
    );

-- Successful task run for task 10013 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12009,
        102,
        1624879944,
        102,
        1624879944,
        11013,
        'Update testdb_dev task run',
        'DONE',
        'bb.task.database.schema.update',
        0,
        '{"detail":"Applied migration version 20210830011437.11013 to database \"testdb_dev\"","migrationId":1,"version":"20210830011437.11013"}',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_dev.tbl2 (name TEXT)"}'
    );

-- Successful task run for task 10014 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        result,
        payload
    )
VALUES
    (
        12010,
        102,
        1624879944,
        102,
        1624879944,
        11014,
        'Update testdb_integration task run',
        'DONE',
        'bb.task.database.schema.update',
        '{"detail":"Applied migration version 20210830011437.11014 to database \"testdb_integration\"","migrationId":1,"version":"20210830011437.11014"}',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_integration.tbl2 (name TEXT)"}'
    );

-- Failed task run for task 10015 create table
INSERT INTO
    task_run (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        status,
        type,
        code,
        result,
        payload
    )
VALUES
    (
        12011,
        102,
        1624879944,
        102,
        1624879944,
        11015,
        'Update testdb_staging task run',
        'FAILED',
        'bb.task.database.schema.update',
        103,
        '{"detail":"table \"tbl2\" already exists"}',
        '{"migrationType":"MIGRATE","statement":"CREATE TABLE testdb_staging.tbl2 (name TEXT)"}'
    );

ALTER SEQUENCE task_run_id_seq RESTART WITH 12012;
