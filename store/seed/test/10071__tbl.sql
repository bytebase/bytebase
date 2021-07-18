-- Table for database 7002 testdb_dev 
INSERT INTO
    tbl (
        id,
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
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
        'OK',
        1625075289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7006,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1625075289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7010,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624475289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7014,
        'tbl1',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624475289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7003,
        'product',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624275289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7007,
        'product',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624575289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7004,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624275289,
        1000,
        4096,
        1024,
        204800,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7004,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624275289,
        1000,
        8192,
        1024,
        204800,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7004,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624275289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7008,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624675289,
        100,
        4096,
        0,
        204800,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7008,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624675289,
        300,
        4096,
        1024,
        102400,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7008,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1624675289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7012,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1622675289,
        200,
        4096,
        1024,
        81920,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7012,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1622675289,
        500,
        4096,
        1024,
        8192,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7012,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1622675289,
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
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7016,
        'user',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1622275289,
        200,
        4096,
        1024,
        0,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7016,
        'post',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1622275289,
        500,
        4096,
        1024,
        2048,
        '',
        ''
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        `type`,
        ENGINE,
        `collation`,
        sync_status,
        last_successful_sync_ts,
        row_count,
        data_size,
        index_size,
        data_free,
        create_options,
        `comment`
    )
VALUES
    (
        101,
        101,
        7016,
        'comment',
        'BASE TABLE',
        'InnoDB',
        'utf8mb4_general_ci',
        'OK',
        1622275289,
        7000,
        8192,
        0,
        8192,
        '',
        ''
    );