-- Activity for issue 13001
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`,
        `level`,
        payload
    )
VALUES
    (
        14001,
        1,
        1,
        13001,
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
        `type`,
        `level`,
        `comment`,
        payload
    )
VALUES
    (
        14002,
        101,
        101,
        13001,
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
        `type`,
        `level`,
        `comment`,
        payload
    )
VALUES
    (
        14003,
        102,
        102,
        13001,
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
        `type`,
        `level`,
        `comment`,
        payload
    )
VALUES
    (
        14004,
        103,
        103,
        13001,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        `type`,
        `level`,
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
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
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
        103,
        1624879944,
        103,
        1624879944,
        13009,
        'bb.issue.create',
        'INFO',
        '',
        '{"issueName":"Create a new table ''tabl2'' using multi-stage SQL review workflow"}'
    );

INSERT INTO
    activity (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11015,"oldStatus":"PENDING","newStatus":"RUNNING","issueName":"Create a new table ''tabl2'' using multi-stage SQL review workflow","taskName":"Update testdb_dev"}'
    );

INSERT INTO
    activity (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        'Applied migration version 20210830011437.11013 to database "testdb_dev"',
        '{"taskId":11015,"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create a new table ''tabl2'' using multi-stage SQL review workflow","taskName":"Update testdb_dev"}'
    );

INSERT INTO
    activity (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        '',
        '{"taskId":11016,"oldStatus":"PENDING","newStatus":"RUNNING","issueName":"Create a new table ''tabl2'' using multi-stage SQL review workflow","taskName":"Update testdb_dev"}'
    );

INSERT INTO
    activity (
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        container_id,
        `type`,
        `level`,
        COMMENT,
        payload
    )
VALUES
    (
        102,
        1624879944,
        102,
        1624879944,
        13009,
        'bb.pipeline.task.status.update',
        'INFO',
        'Applied migration version 20210830020000.11014 to database "testdb_integration"',
        '{"taskId":11016,"oldStatus":"RUNNING","newStatus":"DONE","issueName":"Create a new table ''tabl2'' using multi-stage SQL review workflow","taskName":"Update testdb_integration"}'
    );