-- Task run for task 11002
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624873710,
        1,
        1624873710,
        11002,
        'Add initial schema 1624873710',
        'DONE',
        'bb.task.database.schema.update',
        '',
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"14","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repoFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"da90a2510eccd051ad14e4b89ca904d733169a39","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/da90a2510eccd051ad14e4b89ca904d733169a39","authorName":"tianzhou","added":"bytebase/v1__shop__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11003
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624883710,
        1,
        1624883710,
        11003,
        'Add initial schema 1624883710',
        'DONE',
        'bb.task.database.schema.update',
        '',
        '{"statement":"CREATE TABLE product (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tdescription TEXT NOT NULL\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"14","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repoFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"da90a2510eccd051ad14e4b89ca904d733169a39","title":"Create product table","message":"Create product table","createdTs":1624873354,"url":"http://gitlab.bytebase.com/bytebase-demo/shop/-/commit/da90a2510eccd051ad14e4b89ca904d733169a39","authorName":"tianzhou","added":"bytebase/v1__shop__baseline__add_initial_schema.sql"}}}'
    );

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
        101,
        101,
        11006,
        'Update db1_dev task run',
        'FAILED',
        'bb.task.database.schema.update',
        'table "tbl1" already exists'
    );

-- Task run for task 11008
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624865387,
        1,
        1624865387,
        11008,
        'Add initial schema 1624865387',
        'DONE',
        'bb.task.database.schema.update',
        '',
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","authorName":"tianzhou","added":"bytebase/dev/202106280000__blog__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11009
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624866790,
        1,
        1624866790,
        11009,
        'Add initial schema 1624866790',
        'DONE',
        'bb.task.database.schema.update',
        '',
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","authorName":"tianzhou","added":"bytebase/dev/202106280000__blog__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11010
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624868407,
        1,
        1624868407,
        11010,
        'Add initial schema 1624868407',
        'DONE',
        'bb.task.database.schema.update',
        '',
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","authorName":"tianzhou","added":"bytebase/dev/202106280000__blog__baseline__add_initial_schema.sql"}}}'
    );

-- Task run for task 11011
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624868680,
        1,
        1624868680,
        11011,
        'Add initial schema 1624868680',
        'DONE',
        'bb.task.database.schema.update',
        '',
        '{"statement":"CREATE TABLE `user` (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT NOT NULL\n);\n\nCREATE TABLE post (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tname TEXT NOT NULL,\n\tcontent TEXT NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n\nCREATE TABLE comment (\n\tid INTEGER PRIMARY KEY AUTO_INCREMENT,\n\tcontent TEXT NOT NULL,\n\tpost_id INTEGER NOT NULL,\n\tauthor_id INTEGER NOT NULL,\n\tFOREIGN KEY (post_id) REFERENCES post (id) ON DELETE CASCADE,\n\tFOREIGN KEY (author_id) REFERENCES `user` (id) ON DELETE RESTRICT\n);\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs":1624865383,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/d7f3b88b93c4d7f57b710980cdf92f72dcc4cd1e","authorName":"tianzhou","added":"bytebase/dev/202106280000__blog__baseline__add_initial_schema.sql"}}}'
    );

-- Failed task run for task 11012 create table
INSERT INTO
    task_run (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        task_id,
        name,
        `status`,
        `type`,
        error,
        payload
    )
VALUES
    (
        1,
        1624869944,
        1,
        1624869944,
        11012,
        'Add created at column 1624869944',
        'FAILED',
        'bb.task.database.schema.update',
        'failed to connect instance: On-premises Dev MySQL with user: admin. dial tcp: lookup mysql.dev.example.com: no such host',
        '{"statement":"ALTER TABLE `user` ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE post ADD COLUMN `created_at` DATETIME NOT NULL;\n\nALTER TABLE comment ADD COLUMN `created_at` DATETIME NOT NULL;\n","pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repoId":"13","repoUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repoFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"171ceaf7659ceb8e495aa3ef356ec686656f9dc0","title":"Add created_at column to user,post,comment table for dev environment","message":"Add created_at column to user,post,comment table for dev environment","createdTs":1624869938,"url":"http://gitlab.bytebase.com/bytebase-demo/blog/-/commit/171ceaf7659ceb8e495aa3ef356ec686656f9dc0","authorName":"tianzhou","added":"bytebase/dev/202106280100__blog__add_created_at_column.sql"}}}'
    );