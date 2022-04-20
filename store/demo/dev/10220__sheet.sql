INSERT INTO
    sheet (
        id,
        creator_id,
        updater_id,
        project_id,
        database_id,
        name,
        statement,
        visibility
    )
VALUES
    (
        22001,
        101,
        101,
        3001,
        7019,
        'My test cases',
        'SELECT * FROM test_case',
        'PRIVATE'
    );

INSERT INTO
    sheet (
        id,
        row_status,
        creator_id,
        updater_id,
        project_id,
        name,
        statement,
        visibility
    )
VALUES
    (
        22002,
        'ARCHIVED',
        101,
        101,
        3002,
        'My starred items',
        'SELECT * FROM item WHERE starred=true',
        'PRIVATE'
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
        'All coupon of our shop',
        'SELECT * FROM coupon WHERE shop_id=101',
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
        'All blogs from Alex',
        'SELECT * FROM blog WHERE author="Alex"',
        'PUBLIC'
    );

ALTER SEQUENCE sheet_id_seq RESTART WITH 22005;
