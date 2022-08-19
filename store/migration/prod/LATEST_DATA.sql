-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        101,
        1,
        1,
        'Test',
        0
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        "order"
    )
VALUES
    (
        102,
        1,
        1,
        'Prod',
        1
    );

ALTER SEQUENCE environment_id_seq RESTART WITH 103;

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        101,
        1,
        1,
        101,
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_NEVER"}'
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        102,
        1,
        1,
        102,
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_ALWAYS"}'
    );

INSERT INTO
    policy (
        id,
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        103,
        1,
        1,
        102,
        'bb.policy.backup-plan',
        '{"schedule":"WEEKLY"}'
    );

ALTER SEQUENCE policy_id_seq RESTART WITH 104;

-- Create label keys for `bb.location` and `bb.tenant`.
INSERT INTO
    label_key (
        id,
        creator_id,
        updater_id,
        key
    )
VALUES
    (
        101,
        1,
        1,
        'bb.location'
    );

INSERT INTO
    label_key (
        id,
        creator_id,
        updater_id,
        key
    )
VALUES
    (
        102,
        1,
        1,
        'bb.tenant'
    );

ALTER SEQUENCE label_key_id_seq RESTART WITH 103;

-- Create 1 "test" instance (including * database and admin data source)
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        101,
        1,
        1,
        101,
        'Sample Test instance',
        'POSTGRES',
        '14.3',
        'host.docker.internal',
        '5432',
        ''
    );

-- Create 1 "prod" instance (including * database and admin data source)
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        engine,
        engine_version,
        host,
        port,
        external_link
    )
VALUES
    (
        102,
        1,
        1,
        102,
        'Sample Prod instance',
        'POSTGRES',
        '14.3',
        'host.docker.internal',
        '5433',
        ''
    );

ALTER SEQUENCE instance_id_seq RESTART WITH 103;

-- '*' db for test instance 101
INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        character_set,
        "collation",
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        101,
        1,
        1,
        101,
        1,
        '*',
        'utf8mb4',
        'utf8mb4_general_ci',
        'OK',
        0,
        ''
    );

-- * db for prod instance 102
INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        character_set,
        "collation",
        sync_status,
        last_successful_sync_ts,
        schema_version
    )
VALUES
    (
        102,
        1,
        1,
        102,
        1,
        '*',
        'utf8mb4',
        'utf8mb4_general_ci',
        'OK',
        0,
        ''
    );

ALTER SEQUENCE db_id_seq RESTART WITH 103;

--  admin data source for test instance 101
INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        password
    )
VALUES
    (
        101,
        1,
        1,
        101,
        101,
        'Admin data source',
        'ADMIN',
        'root',
        ''
    );

-- admin data source for prod instance 102
INSERT INTO
    data_source (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        name,
        TYPE,
        username,
        password
    )
VALUES
    (
        102,
        1,
        1,
        102,
        102,
        'Admin data source',
        'ADMIN',
        'root',
        ''
    );

ALTER SEQUENCE data_source_id_seq RESTART WITH 103;

-- Create pipeline/stage/task/issue for onboarding
-- Create pipeline 101 "Hello world"
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        status
    )
VALUES
    (
        101,
        1,
        1,
        'Pipeline - Hello world',
        'OPEN'
    );

ALTER SEQUENCE pipeline_id_seq RESTART WITH 102;

-- Create stage 101, 102 for pipeline 101
INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        101,
        1,
        1,
        101,
        101,
        'Test'
    );

INSERT INTO
    stage (
        id,
        creator_id,
        updater_id,
        pipeline_id,
        environment_id,
        name
    )
VALUES
    (
        102,
        1,
        1,
        101,
        102,
        'Prod'
    );

ALTER SEQUENCE stage_id_seq RESTART WITH 103;

-- Create task 101 for stage 101
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
        101,
        1,
        1,
        101,
        101,
        101,
        NULL,
        'Welcome',
        'bb.task.general',
        'RUNNING',
        '{"statement":"SELECT ''Welcome Builders'';"}'
    );

-- Create task 102 for stage 102
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
        102,
        1,
        1,
        101,
        102,
        102,
        NULL,
        'Let''s go',
        'bb.task.general',
        'PENDING_APPROVAL',
        '{"statement":"SELECT ''Let''s go'';"}'
    );

ALTER SEQUENCE task_id_seq RESTART WITH 103;

-- Create task_run for task 101
INSERT INTO
    task_run (
        id,
        creator_id,
        updater_id,
        task_id,
        name,
        status,
        type,
        comment,
        result,
        payload
    )
VALUES
    (
        101,
        1,
        1,
        101,
        'Welcome',
        'FAILED',
        'bb.task.general',
        '',
        '{"detail":"Something is not right..."}',
        '{}'
    );

INSERT INTO
    task_run (
        id,
        creator_id,
        updater_id,
        task_id,
        name,
        status,
        type,
        comment,
        payload
    )
VALUES
    (
        102,
        1,
        1,
        101,
        'Welcome',
        'RUNNING',
        'bb.task.general',
        'Let''s give another try',
        '{}'
    );

ALTER SEQUENCE task_run_id_seq RESTART WITH 103;

-- Create issue 101 "Hello world"
INSERT INTO
    issue (
        id,
        creator_id,
        updater_id,
        project_id,
        pipeline_id,
        name,
        status,
        type,
        description,
        assignee_id
    )
VALUES
    (
        101,
        1,
        1,
        1,
        101,
        'Hello world!',
        'OPEN',
        'bb.issue.general',
        'Welcome to Bytebase, this is the issue interface where developers and DBAs collaborate on database schema management issues such as: '||chr(10)||' - Creating a new database'||chr(10)||' - Creating a table'||chr(10)||' - Creating an index'||chr(10)||' - Adding/Altering a column'||chr(10)||' - Troubleshooting performance issue'||chr(10)||'Let''s try some simple tasks:'||chr(10)||'1. Bookmark this issue by clicking the star icon on the top of this page'||chr(10)||'2. Leave a comment below to greet future customers.',
        1
    );

ALTER SEQUENCE issue_id_seq RESTART WITH 102;

-- Create activity 101
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level
    )
VALUES
    (
        101,
        1,
        1,
        101,
        'bb.issue.create',
        'INFO'
    );

INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        type,
        level,
        comment
    )
VALUES
    (
        102,
        1,
        1,
        101,
        'bb.issue.comment.create',
        'INFO',
        'Go fish!'
    );

ALTER SEQUENCE activity_id_seq RESTART WITH 103;
