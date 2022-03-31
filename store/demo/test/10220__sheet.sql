-- Sheet for principal 101 "Demo Owner"
INSERT INTO
    sheet (
        id,
        creator_id,
        updater_id,
        project_id,
        database_id,
        name,
        statement,
        source,
        type
    )
VALUES
    (
        22001,
        101,
        101,
        3001,
        7019,
        'my-sheet',
        'SELECT * FROM demo',
        'BYTEBASE',
        'SQL'
    );

ALTER SEQUENCE sheet_id_seq RESTART WITH 22002;
