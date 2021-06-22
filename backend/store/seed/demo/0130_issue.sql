-- Issue for the single stage, single step "Hello world" pipeline
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
        assignee_id,
        subscriber_id_list
    )
VALUES
    (
        13001,
        102,
        102,
        3001,
        9001,
        'Hello world!',
        'OPEN',
        'bb.issue.general',
        'Welcome to Bytebase, this is the issue interface where tech leads, developers and DBAs collaborate on database management issues such as: ' || char(10, 10) || ' - Requesting a new database' || char(10) || ' - Creating a table' || char(10) || ' - Creating an index' || char(10) || ' - Adding a column' || char(10) || ' - Troubleshooting performance issue' || char(10, 10) || 'Let''s bookmark this issue by clicking the star icon on the top of this page.' || char(10, 10) || 'And then click the ''Approve'' button on the top right.',
        101,
        '101,102,103,104'
    );

-- Issue for the multi stage add column pipeline
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
        assignee_id,
        subscriber_id_list
    )
VALUES
    (
        13002,
        103,
        103,
        3001,
        9002,
        'Add column ''location'' to table ''warehouse''',
        'OPEN',
        'bb.issue.database.schema.update',
        'Add the location column to record the warehouse address.',
        101,
        '101,102,103,104'
    );

-- Issue for the multi stage create table pipeline
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
        assignee_id,
        subscriber_id_list
    )
VALUES
    (
        13003,
        103,
        103,
        3002,
        9003,
        'Create a new table ''tbl1''',
        'OPEN',
        'bb.issue.database.schema.update',
        'Create tbl1.',
        101,
        '101,102,103,104'
    );

-- Issue for simulating webhook push event
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
        assignee_id,
        subscriber_id_list
    )
VALUES
    (
        13004,
        1,
        1,
        3001,
        9004,
        'Create todo table to stagingdb1',
        'OPEN',
        'bb.issue.database.schema.update',
        'Create todo table to stagingdb1',
        1,
        ''
    );