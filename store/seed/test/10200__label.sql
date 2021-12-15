-- Label key for `bb.location`.
INSERT INTO
    label_key (
        id,
        creator_id,
        updater_id,
        key
    )
VALUES
    (
        20001,
        1,
        1,
        'bb.location'
    );

-- Label key for `bb.tenant`.
INSERT INTO
    label_key (
        id,
        creator_id,
        updater_id,
        key
    )
VALUES
    (
        20002,
        1,
        1,
        'bb.tenant'
    );

-- Label values.
INSERT INTO
    label_value (
        id,
        creator_id,
        updater_id,
        key,
        value
    )
VALUES
    (
        20010,
        1,
        1,
        "bb.location",
        "earth"
    );

INSERT INTO
    label_value (
        id,
        creator_id,
        updater_id,
        key,
        value
    )
VALUES
    (
        20011,
        1,
        1,
        "bb.tenant",
        "bytebase"
    );

-- Database labels.
INSERT INTO
    db_label (
        id,
        creator_id,
        updater_id,
        database_id,
        key,
        value
    )
VALUES
    (
        20021,
        1,
        1,
        7002,
        "bb.location",
        "earth"
    );

INSERT INTO
    db_label (
        id,
        creator_id,
        updater_id,
        database_id,
        key,
        value
    )
VALUES
    (
        20022,
        1,
        1,
        7002,
        "bb.tenant",
        "bytebase"
    );
