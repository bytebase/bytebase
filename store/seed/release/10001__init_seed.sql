-- Create "test" and "prod" environments
INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`
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
    policy (
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        1,
        1,
        101,
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_NEVER"}'
    );

INSERT INTO
    environment (
        id,
        creator_id,
        updater_id,
        name,
        `order`
    )
VALUES
    (
        102,
        1,
        1,
        'Prod',
        1
    );

INSERT INTO
    policy (
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        1,
        1,
        102,
        'bb.policy.pipeline-approval',
        '{"value":"MANUAL_APPROVAL_ALWAYS"}'
    );

INSERT INTO
    policy (
        creator_id,
        updater_id,
        environment_id,
        type,
        payload
    )
VALUES
    (
        1,
        1,
        102,
        'bb.policy.backup-plan',
        '{"schedule":"WEEKLY"}'
    );

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

-- Create 1 "test", 1 "prod" instance (including * database and admin data source)
-- Both instances contains the connection info we expect user to setup according to https://docs.bytebase.com/install/docker#start-a-mysql-docker-instance-for-testing
-- Set host to 172.17.0.1 which is the default docker gateway ip.
-- Our quickstart guide suggests to run both Bytebase and MySQL using docker, and in such case, bytebase access the mysqld container via 172.17.0.1
-- "test" instance
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
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
        'MySQL Test (Follow the "External Link" field to bring up the MySQL server)',
        'MYSQL',
        '8.0.19',
        'host.docker.internal',
        '3306',
        'https://docs.bytebase.com/install/docker#start-a-mysql-docker-instance-for-testing'
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        character_set,
        `collation`,
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
        `password`
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
        'testpwd1'
    );

-- "prod" instance
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
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
        'MySQL Prod (Follow the "External Link" field to bring up the MySQL server)',
        'MYSQL',
        '8.0.19',
        'host.docker.internal',
        '3306',
        'https://docs.bytebase.com/install/docker#start-a-mysql-docker-instance-for-testing'
    );

INSERT INTO
    db (
        id,
        creator_id,
        updater_id,
        instance_id,
        project_id,
        name,
        character_set,
        `collation`,
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
        `password`
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
        'testpwd1'
    );

-- Create pipeline/stage/task/issue for onboarding
-- Create pipeline 101 "Hello world"
INSERT INTO
    pipeline (
        id,
        creator_id,
        updater_id,
        name,
        `status`
    )
VALUES
    (
        101,
        1,
        1,
        'Pipeline - Hello world',
        'OPEN'
    );

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
        `type`,
        `status`,
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
        `type`,
        `status`,
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

-- Create task_run for task 101
INSERT INTO
    task_run (
        creator_id,
        updater_id,
        task_id,
        name,
        `status`,
        `type`,
        comment,
        result,
        payload
    )
VALUES
    (
        1,
        1,
        101,
        'Welcome',
        'FAILED',
        'bb.task.general',
        '',
        '{"detail":"Something is not right..."}',
        ''
    );

INSERT INTO
    task_run (
        creator_id,
        updater_id,
        task_id,
        name,
        `status`,
        `type`,
        comment,
        payload
    )
VALUES
    (
        1,
        1,
        101,
        'Welcome',
        'RUNNING',
        'bb.task.general',
        'Let''s give another try',
        ''
    );

-- Create issue 101 "Hello world"
INSERT INTO
    issue (
        id,
        creator_id,
        updater_id,
        project_id,
        pipeline_id,
        name,
        `status`,
        `type`,
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
        'Welcome to Bytebase, this is the issue interface where developers and DBAs collaborate on database schema management issues such as: ' || char(10, 10) || ' - Creating a new database' || char(10) || ' - Creating a table' || char(10) || ' - Creating an index' || char(10) || ' - Adding/altering a column' || char(10) || ' - Troubleshooting performance issue' || char(10, 10) || 'Let''s try some simple tasks:' || char(10, 10) || '1. Bookmark this issue by clicking the star icon on the top of this page' || char(10) || '2. Leave a comment below to greet future comers' || char(10) || '3. Follow the Quickstart on the bottom left to get familiar with other features',
        1
    );

-- Create activity 101
INSERT INTO
    activity (
        id,
        creator_id,
        updater_id,
        container_id,
        `type`,
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
        creator_id,
        updater_id,
        container_id,
        `type`,
        level,
        `comment`
    )
VALUES
    (
        1,
        1,
        101,
        'bb.issue.comment.create',
        'INFO',
        'Go fish!'
    );
