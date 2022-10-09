-- Table for database 7002 testdb_dev
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7101,
        101,
        101,
        7002,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1234,
        16384,
        0,
        0,
        '',
        ''
    );

-- Table for database 7006 testdb_integration
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7102,
        101,
        101,
        7006,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1234,
        16384,
        16384,
        65536,
        '',
        ''
    );

-- Table for database 7010 testdb_staging
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7103,
        101,
        101,
        7010,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        12234,
        65536,
        16384,
        655360,
        '',
        ''
    );

-- Table for database 7014 testdb_prod
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7104,
        101,
        101,
        7014,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1234,
        32768,
        4096,
        327680,
        '',
        ''
    );

-- Table for database 7003 shop
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7105,
        101,
        101,
        7003,
        'product',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1000,
        4096,
        0,
        409600,
        '',
        ''
    );

-- Table for database 7007 shop
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7106,
        101,
        101,
        7007,
        'product',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1000,
        4096,
        1024,
        0,
        '',
        ''
    );

-- Table for database 7004 blog
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7107,
        101,
        101,
        7004,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1000,
        4096,
        1024,
        204800,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7108,
        101,
        101,
        7004,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        1000,
        8192,
        1024,
        204800,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7109,
        101,
        101,
        7004,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        30000,
        65536,
        4096,
        102400,
        '',
        ''
    );

-- Table for database 7008 blog
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7110,
        101,
        101,
        7008,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        100,
        4096,
        0,
        204800,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7111,
        101,
        101,
        7008,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        300,
        4096,
        1024,
        102400,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7112,
        101,
        101,
        7008,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        2000,
        65536,
        2048,
        40960,
        '',
        ''
    );

-- Table for database 7012 blog
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7113,
        101,
        101,
        7012,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        200,
        4096,
        1024,
        81920,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7114,
        101,
        101,
        7012,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        500,
        4096,
        1024,
        8192,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7115,
        101,
        101,
        7012,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        4000,
        65536,
        65536,
        20480,
        '',
        ''
    );

-- Table for database 7016 blog
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7116,
        101,
        101,
        7016,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        200,
        4096,
        1024,
        0,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7117,
        101,
        101,
        7016,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        500,
        4096,
        1024,
        2048,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7118,
        101,
        101,
        7016,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        7000,
        8192,
        0,
        8192,
        '',
        ''
    );

INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        type,
        ENGINE,
        "collation",
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        comment
    )
VALUES
    (
        7119,
        101,
        101,
        7019,
        'blog',
        'BASE TABLE',
        -- PostgreSQL doesn't use pluggable engines.
        'ACID',
        'utf8mb4_general_ci',
        7000,
        8192,
        0,
        8192,
        '',
        ''
    );

ALTER SEQUENCE tbl_id_seq RESTART WITH 7120;
