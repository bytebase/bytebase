INSERT INTO
    sheet_organizer (
        id,
        sheet_id,
        principal_id,
        starred,
        pinned
    )
VALUES
    (
        23001,
        22001,
        101,
        false,
        false
    );

INSERT INTO
    sheet_organizer (
        id,
        sheet_id,
        principal_id,
        starred,
        pinned
    )
VALUES
    (
        23002,
        22002,
        101,
        true,
        false
    );

INSERT INTO
    sheet_organizer (
        id,
        sheet_id,
        principal_id,
        starred,
        pinned
    )
VALUES
    (
        23003,
        22003,
        101,
        true,
        true
    );

ALTER SEQUENCE sheet_id_seq RESTART WITH 23004;
