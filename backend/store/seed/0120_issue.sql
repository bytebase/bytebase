-- Issue for the single stage, single step "Hello world" pipeline
INSERT INTO
    issue (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        pipeline_id,
        name,
        `status`,
        `type`,
        description,
        assignee_id,
        subscriber_id_list,
        `sql`,
        rollback_sql
    )
VALUES
    (
        12001,
        1002,
        1002,
        1,
        3001,
        9001,
        'Hello world!',
        'OPEN',
        'bb.general',
        'Welcome to Bytebase, this is the issue interface where tech leads, developers and DBAs collaborate on database management issues such as: ' || char(10, 10) || ' - Requesting a new database' || char(10) || ' - Creating a table' || char(10) || ' - Creating an index' || char(10) || ' - Adding a column' || char(10) || ' - Troubleshooting performance issue' || char(10, 10) || 'Let''s bookmark this issue by clicking the star icon on the top of this page.',
        1001,
        '1001,1002,1003,1004',
        'SELECT ''Welcome''' || char(10) || 'FROM engineering' || char(10) || 'WHERE role IN (''Tech Lead'', ''Developer'', ''DBA'')' || char(10) || 'AND taste = ''Good'';',
        ''
    );

-- Issue for the multi stage update schema pipeline
INSERT INTO
    issue (
        id,
        creator_id,
        updater_id,
        workspace_id,
        project_id,
        pipeline_id,
        name,
        `status`,
        `type`,
        description,
        assignee_id,
        subscriber_id_list,
        `sql`,
        rollback_sql
    )
VALUES
    (
        12002,
        1003,
        1003,
        1,
        3002,
        9002,
        'Add column ''location'' to table ''warehouse''',
        'OPEN',
        'bb.database.schema.update',
        'Add the location column to record the warehouse address.',
        1001,
        '1001,1002,1003,1004',
        'ALTER TABLE warehouse ' || char(10) || 'ADD COLUMN location VARCHAR(255);',
        'ALTER TABLE warehouse ' || char(10) || 'DROP COLUMN location;'
    );