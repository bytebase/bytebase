-- Label keys.
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

ALTER SEQUENCE label_key_id_seq RESTART WITH 20003;

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
        'bb.location',
        'earth'
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
        'bb.tenant',
        'bytebase'
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
        20012,
        1,
        1,
        'bb.location',
        'east'
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
        20013,
        1,
        1,
        'bb.location',
        'west'
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
        20014,
        1,
        1,
        'bb.tenant',
        'turtle_rock'
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
        20015,
        1,
        1,
        'bb.tenant',
        'echo_island'
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
        20016,
        1,
        1,
        'bb.tenant',
        'public_test'
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
        20017,
        1,
        1,
        'bb.tenant',
        'dev'
    );

ALTER SEQUENCE label_value_id_seq RESTART WITH 20018;

-- Database labels for database in tenant mode disabled project.
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
        'bb.location',
        'earth'
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
        'bb.tenant',
        'bytebase'
    );

-- Database labels for database in tenant mode project.
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
        20023,
        1,
        1,
        7021,
        'bb.location',
        'earth'
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
        20024,
        1,
        1,
        7022,
        'bb.tenant',
        'bytebase'
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
        20025,
        1,
        1,
        7023,
        'bb.location',
        'earth'
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
        20026,
        1,
        1,
        7024,
        'bb.tenant',
        'bytebase'
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
        20027,
        1,
        1,
        7027,
        'bb.tenant',
        'dev'
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
        20028,
        1,
        1,
        7028,
        'bb.tenant',
        'public_test'
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
        20029,
        1,
        1,
        7029,
        'bb.tenant',
        'turtle_rock'
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
        20030,
        1,
        1,
        7029,
        'bb.location',
        'east'
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
        20031,
        1,
        1,
        7030,
        'bb.tenant',
        'echo_island'
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
        20032,
        1,
        1,
        7030,
        'bb.location',
        'west'
    );

ALTER SEQUENCE db_label_id_seq RESTART WITH 20033;
