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
        22101,
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
        22102,
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
        22103,
        22003,
        102,
        false,
        true
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
        22104,
        22004,
        102,
        true,
        false
    );

ALTER SEQUENCE sheet_id_seq RESTART WITH 22105;
