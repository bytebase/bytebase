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

-- Create 1 "test" and 1 "prod" instance (including * database and admin data source)
-- "test" instance
INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
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
        'On-premises Test MySQL',
        'MYSQL',
        'mysql.test.example.com',
        '3306',
        'bytebase.com'
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
        last_successful_sync_ts
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
        'utf8mb4_0900_ai_ci',
        'OK',
        0
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
        PASSWORD
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
        'admin',
        ''
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
        'On-premises Prod MySQL',
        'MYSQL',
        'mysql.prod.example.com',
        '3306',
        'bytebase.com'
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
        last_successful_sync_ts
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
        'utf8mb4_0900_ai_ci',
        'OK',
        0
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
        PASSWORD
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
        'admin',
        ''
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
        detail,
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
        'Something is not right...',
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
        detail,
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
        `type`
    )
VALUES
    (
        101,
        1,
        1,
        101,
        'bb.issue.create'
    );

INSERT INTO
    activity (
        creator_id,
        updater_id,
        container_id,
        `type`,
        `comment`
    )
VALUES
    (
        1,
        1,
        101,
        'bb.issue.comment.create',
        'Go fish!'
    );