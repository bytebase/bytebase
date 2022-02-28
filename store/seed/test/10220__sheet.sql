-- Sheet for principal 101 "Demo Owner"
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
        22001,
        101,
        101,
        3001,
        'my-sheet',
        'SELECT * FROM demo'
    );

ALTER SEQUENCE sheet_id_seq RESTART WITH 22002;
