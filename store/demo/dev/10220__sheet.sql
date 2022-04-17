-- Sheet for principal 101 "Demo Owner"
INSERT INTO
    sheet (
        id,
        creator_id,
        updater_id,
        project_id,
        database_id,
        name,
        statement
    )
VALUES
    (
        22001,
        101,
        101,
        3001,
        7019,
        'demo data',
        'SELECT * FROM demo'
    );

INSERT INTO
    sheet (
        id,
        creator_id,
        updater_id,
        project_id,
        name,
        statement
    )
VALUES
    (
        22002,
        101,
        101,
        3002,
        'all employee',
        'SELECT * FROM employee'
    );

INSERT INTO
    sheet (
        id,
        creator_id,
        updater_id,
        project_id,
        name,
        statement,
        visibility
    )
VALUES
    (
        22003,
        102,
        102,
        3002,
        'shared employee with project',
        'SELECT * FROM employee',
        'PROJECT'
    );

INSERT INTO
    sheet (
        id,
        creator_id,
        updater_id,
        project_id,
        name,
        statement,
        visibility
    )
VALUES
    (
        22004,
        103,
        103,
        3003,
        'shared all employee',
        'SELECT * FROM employee',
        'PUBLIC'
    );

ALTER SEQUENCE sheet_id_seq RESTART WITH 22005;
