-- Activity for issue 101
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        payload
    )
VALUES
    (
        14001,
        1,
        1,
        101,
        'bb.issue.create',
        'INFO',
        '{"issueName":"Hello world!"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        comment,
        payload
    )
VALUES
    (
        14002,
        101,
        101,
        101,
        'bb.issue.comment.create',
        'INFO',
        'Welcome!',
        '{"issueName":"Hello world!"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        comment,
        payload
    )
VALUES
    (
        14003,
        102,
        102,
        101,
        'bb.issue.comment.create',
        'INFO',
        'Let''s rock!',
        '{"issueName":"Hello world!"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        comment,
        payload
    )
VALUES
    (
        14004,
        103,
        103,
        101,
        'bb.issue.comment.create',
        'INFO',
        'Go fish!',
        '{"issueName":"Hello world!"}'
    );

-- Activity for issue 13002
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        payload
    )
VALUES
    (
        14005,
        1,
        1624873710,
        1,
        1624873710,
        13002,
        'bb.issue.create',
        'INFO',
        '{"issueName":"Create product table"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14006,
        1,
        1624873710,
        1,
        1624873710,
        13002,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11002,"oldStatus":"PENDING","newStatus":"RUNNING","issueName":"Create product table","taskName":"Add initial schema"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14007,
        1,
        1624873710,
        1,
        1624873710,
        13002,
        'bb.pipeline.task.status.update',
        'INFO',
        'Established baseline version 202106280000 for database ''shop''',
        '{"taskId":11002,"oldStatus":"RUNNING","newStatus":"DONE"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14008,
        1,
        1624873710,
        1,
        1624873710,
        13002,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11003,"oldStatus":"PENDING","newStatus":"RUNNING"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14009,
        1,
        1624873710,
        1,
        1624873710,
        13002,
        'bb.pipeline.task.status.update',
        'INFO',
        'Established baseline version 202106280000 for database ''shop''',
        '{"taskId":11003,"oldStatus":"RUNNING","newStatus":"DONE"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        payload
    )
VALUES
    (
        14010,
        1,
        1624873710,
        1,
        1624873710,
        13002,
        'bb.issue.field.update',
        'INFO',
        '{"fieldId":"3","oldValue":"1","newValue":"101","issueName":"Create product table"}'
    );

-- Activity for issue 13003
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        payload
    )
VALUES
    (
        14011,
        103,
        1624873710,
        103,
        1624873710,
        13003,
        'bb.issue.create',
        'INFO',
        '{"issueName":"CREATE a new TABLE ''tbl1''"}'
    );

-- Activity for failed task_run 12001
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14012,
        1,
        1,
        13003,
        'bb.pipeline.task.status.update',
        'ERROR',
        'table "tbl1" already exists',
        '{"taskId":11006,"oldStatus":"RUNNING","newStatus":"FAILED","issueName":"Create a new table ''tbl1''","taskName":"Update testdb_dev"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        payload
    )
VALUES
    (
        14013,
        102,
        102,
        13003,
        'bb.issue.status.update',
        'INFO',
        '{"oldStatus":"OPEN","newStatus":"CANCELED","issueName":"Create a new table ''tbl1''"}'
    );

-- Activity for issue 13004
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14014,
        1,
        1624865387,
        1,
        1624865387,
        13004,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Create user, post, comment table for dev environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14015,
        1,
        1624865388,
        1,
        1624865388,
        13004,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11008,"oldStatus":"PENDING","newStatus":"RUNNING"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14016,
        1,
        1624865388,
        1,
        1624865388,
        13004,
        'bb.pipeline.task.status.update',
        'INFO',
        'Established baseline version 202106280000 for database ''blog''',
        '{"taskId":11008,"oldStatus":"RUNNING","newStatus":"DONE"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14017,
        1,
        1624865388,
        1,
        1624865388,
        13004,
        'bb.issue.status.update',
        'INFO',
        '',
        '{"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create user, post, comment table for dev environment"}'
    );

-- Activity for issue 13005
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14018,
        1,
        1624866790,
        1,
        1624866790,
        13005,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Create user, post, comment table for integration environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14019,
        1,
        1624866791,
        1,
        1624866791,
        13005,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11009,"oldStatus":"PENDING","newStatus":"RUNNING"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14020,
        1,
        1624866791,
        1,
        1624866791,
        13005,
        'bb.pipeline.task.status.update',
        'INFO',
        'Established baseline version 202106280000 for database ''blog''',
        '{"taskId":11009,"oldStatus":"RUNNING","newStatus":"DONE"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14021,
        1,
        1624866791,
        1,
        1624866791,
        13005,
        'bb.issue.status.update',
        'INFO',
        '',
        '{"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create user, post, comment table for integration environment"}'
    );

-- Activity for issue 13006
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14022,
        1,
        1624868407,
        1,
        1624868407,
        13006,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Create user, post, comment table for staging environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14023,
        1,
        1624868408,
        1,
        1624868408,
        13006,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11010,"oldStatus":"PENDING","newStatus":"RUNNING"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14024,
        1,
        1624868408,
        1,
        1624868408,
        13006,
        'bb.pipeline.task.status.update',
        'INFO',
        'Established baseline version 202106280000 for database ''blog''',
        '{"taskId":11010,"oldStatus":"RUNNING","newStatus":"DONE"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14025,
        1,
        1624868408,
        1,
        1624868408,
        13006,
        'bb.issue.status.update',
        'INFO',
        '',
        '{"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create user, post, comment table for staging environment"}'
    );

-- Activity for issue 13007
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14026,
        1,
        1624868680,
        1,
        1624868680,
        13007,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Create user, post, comment table for prod environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14027,
        1,
        1624868681,
        1,
        1624868681,
        13007,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11011,"oldStatus":"PENDING","newStatus":"RUNNING"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14028,
        1,
        1624868681,
        1,
        1624868681,
        13007,
        'bb.pipeline.task.status.update',
        'INFO',
        'Established baseline version 202106280000 for database ''blog''',
        '{"taskId":11011,"oldStatus":"RUNNING","newStatus":"DONE"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14029,
        1,
        1624868681,
        1,
        1624868681,
        13007,
        'bb.issue.status.update',
        'INFO',
        '',
        '{"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create user, post, comment table for prod environment"}'
    );

-- Activity for issue 13008
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14030,
        1,
        1624869944,
        1,
        1624869944,
        13008,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Add created_at column to user,post,comment table for dev environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14031,
        1,
        1624869945,
        1,
        1624869945,
        13008,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11012,"oldStatus":"PENDING","newStatus":"RUNNING"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14032,
        1,
        1624869945,
        1,
        1624869945,
        13008,
        'bb.pipeline.task.status.update',
        'ERROR',
        'database ''blog'' has already applied version 202106280100',
        '{"taskId":11012,"oldStatus":"RUNNING","newStatus":"FAILED"}'
    );

-- Activity for issue 13009
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14033,
        103,
        1624879944,
        103,
        1624879944,
        13009,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14034,
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11015,"oldStatus":"PENDING","newStatus":"RUNNING","issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow","taskName":"Update testdb_dev"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14035,
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        'Applied migration version 20210830011437.11013 to database "testdb_dev"',
        '{"taskId":11013,"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow","taskName":"Update testdb_dev"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14036,
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11014,"oldStatus":"PENDING","newStatus":"RUNNING","issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow","taskName":"Update testdb_integration"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14037,
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        'Applied migration version 20210830011437.11014 to database "testdb_integration"',
        '{"taskId":11014,"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow","taskName":"Update testdb_integration"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14038,
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11015,"oldStatus":"PENDING","newStatus":"RUNNING","issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow","taskName":"Update testdb_staging"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14039,
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'ERROR',
        'table "tbl2" already exists',
        '{"taskId":11015,"oldStatus":"RUNNING","newStatus":"FAILED","issueName":"Create a new table ''tbl2'' using multi-stage SQL review workflow","taskName":"Update testdb_staging"}'
    );

-- Project activity for 3001
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14040,
        102,
        1624879944,
        102,
        1624879944,
        3001,
        'bb.project.member.role.update',
        'INFO',
        'Changed Demo Owner (demo@example.com) from OWNER to DEVELOPER.',
        '{}'
    );

-- Project activity for 3002
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14041,
        101,
        1624879944,
        101,
        1624879944,
        3002,
        'bb.project.member.create',
        'INFO',
        'Granted Jerry DBA to jerry@example.com (DEVELOPER).',
        '{}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14042,
        1,
        1624879944,
        1,
        1624879944,
        3002,
        'bb.project.repository.push',
        'INFO',
        'Created issue "Create product table using multi-stage VCS workflow".',
        '{"pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"14","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/shop","repositoryFullPath":"bytebase-demo/shop","authorName":"tianzhou","fileCommit":{"id":"c80352facbaefcde0c1c82340381be3286e5438d","title":"Create product table","message":"Create product table","createdTs": 1630940811,"url":"https://gitlab.bytebase.com/bytebase-demo/shop/-/commit/c80352facbaefcde0c1c82340381be3286e5438d","authorName":"tianzhou","added":"bytebase/shop__v1__baseline__add_initial_schema.sql"}},"issueId":13002,"issueName":"Create product table using multi-stage VCS workflow"}'
    );

-- Project activity for 3003
INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14043,
        101,
        1624879944,
        101,
        1624879944,
        3003,
        'bb.project.member.create',
        'INFO',
        'Granted Tom Dev to tom@example.com (DEVELOPER).',
        '{}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14044,
        1,
        1624879944,
        1,
        1624879944,
        3003,
        'bb.project.repository.push',
        'INFO',
        'Created issue "Add created_at column to user,post,comment table for dev environment".',
        '{"pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8ff6482998059a861e1faa14658c65244577b54e","title":"Add created_at column to user,post,comment table for dev environment","message":"Add created_at column to user,post,comment table for dev environment","createdTs":1630943211,"url":"https://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8ff6482998059a861e1faa14658c65244577b54e","authorName":"tianzhou","added":"bytebase/dev/blog__202106280100__migrate__add_created_at_column.sql"}},"issueId":13008,"issueName":"Add created_at column to user,post,comment table for dev environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14045,
        1,
        1624879944,
        1,
        1624879944,
        3003,
        'bb.project.repository.push',
        'INFO',
        'Created issue "Create user, post, comment table for dev environment".',
        '{"pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"8f578d53e821c46421d69fd0aabd29921190a6c0","title":"Create user, post, comment table for dev environment","message":"Create user, post, comment table for dev environment","createdTs": 1630941711,"url":"https://gitlab.bytebase.com/bytebase-demo/blog/-/commit/8f578d53e821c46421d69fd0aabd29921190a6c0","authorName":"tianzhou","added":"bytebase/dev/blog__202106280000__baseline__add_initial_schema.sql"}},"issueId":13004,"issueName":"Create user, post, comment table for dev environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14046,
        1,
        1624879944,
        1,
        1624879944,
        3003,
        'bb.project.repository.push',
        'INFO',
        'Created issue "Create user, post, comment table for integration environment".',
        '{"pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"0765950c28c50dd07217aa0a133e8239e64f0ea1","title":"Create user, post, comment table for integration environment","message":"Create user, post, comment table for integration environment","createdTs": 1630941711,"url":"https://gitlab.bytebase.com/bytebase-demo/blog/-/commit/0765950c28c50dd07217aa0a133e8239e64f0ea1","authorName":"tianzhou","added":"bytebase/integration/blog__202106280000__baseline__add_initial_schema.sql"}},"issueId":13005,"issueName":"Create user, post, comment table for integration environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14047,
        1,
        1624879944,
        1,
        1624879944,
        3003,
        'bb.project.repository.push',
        'INFO',
        'Created issue "Create user, post, comment table for staging environment".',
        '{"pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"149f18b8d94805be4b92e256ef1dc9f4a27d5157","title":"Create user, post, comment table for staging environment","message":"Create user, post, comment table for staging environment","createdTs": 1630941711,"url":"https://gitlab.bytebase.com/bytebase-demo/blog/-/commit/149f18b8d94805be4b92e256ef1dc9f4a27d5157","authorName":"tianzhou","added":"bytebase/staging/blog__202106280000__baseline__add_initial_schema.sql"}},"issueId":13006,"issueName":"Create user, post, comment table for staging environment"}'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        type,
        level,
        COMMENT,
        payload
    )
VALUES
    (
        14048,
        1,
        1624879944,
        1,
        1624879944,
        3003,
        'bb.project.repository.push',
        'INFO',
        'Created issue "Create user, post, comment table for prod environment".',
        '{"pushEvent":{"vcsType":"GITLAB_SELF_HOST","baseDir":"bytebase","ref":"refs/heads/master","repositoryId":"13","repositoryUrl":"http://gitlab.bytebase.com/bytebase-demo/blog","repositoryFullPath":"bytebase-demo/blog","authorName":"tianzhou","fileCommit":{"id":"ec17d38d6581015feb49b341cc3da56cb0e354fb","title":"Create user, post, comment table for prod environment","message":"Create user, post, comment table for prod environment","createdTs": 1630941711,"url":"https://gitlab.bytebase.com/bytebase-demo/blog/-/commit/ec17d38d6581015feb49b341cc3da56cb0e354fb","authorName":"tianzhou","added":"bytebase/prod/blog__202106280000__baseline__add_initial_schema.sql"}},"issueId":13006,"issueName":"Create user, post, comment table for prod environment"}'
    );

ALTER SEQUENCE activity_id_seq RESTART WITH 14049;
