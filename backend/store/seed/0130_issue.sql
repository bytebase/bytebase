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
        `sql`
    )
VALUES
    (
        13001,
        1002,
        1002,
        1,
        3001,
        9001,
        'Hello world!',
        'OPEN',
        'bb.issue.general',
        'Welcome to Bytebase, this is the issue interface where tech leads, developers and DBAs collaborate on database management issues such as: ' || char(10, 10) || ' - Requesting a new database' || char(10) || ' - Creating a table' || char(10) || ' - Creating an index' || char(10) || ' - Adding a column' || char(10) || ' - Troubleshooting performance issue' || char(10, 10) || 'Let''s bookmark this issue by clicking the star icon on the top of this page.' || char(10, 10) || 'And then click the ''Approve'' button on the top right.',
        1001,
        '1001,1002,1003,1004',
        'SELECT ''Welcome Tech Lead, DBA, Developer'';'
    );

-- Issue for the multi stage add column pipeline
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
        13002,
        1003,
        1003,
        1,
        3002,
        9002,
        'Add column ''location'' to table ''warehouse''',
        'OPEN',
        'bb.issue.db.schema.update',
        'Add the location column to record the warehouse address.',
        1001,
        '1001,1002,1003,1004',
        'ALTER TABLE warehouse ' || char(10) || 'ADD COLUMN location VARCHAR(255);',
        'ALTER TABLE warehouse ' || char(10) || 'DROP COLUMN location;'
    );

-- Issue for the multi stage create table pipeline
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
        13003,
        1003,
        1003,
        1,
        3002,
        9003,
        'Create a new table ''tbl1''',
        'OPEN',
        'bb.issue.db.schema.update',
        'Create tbl1.',
        1001,
        '1001,1002,1003,1004',
        'CREATE TABLE tbl1;',
        'DROP TABLE tbl1;'
    );