-- Table for database 7002 testdb_dev 
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7002,
        'tbl1',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1234,
        16384,
        0,
        'OK',
        1625075289
    );

-- Table for database 7006 testdb_integration 
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7006,
        'tbl1',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1234,
        16384,
        16384,
        'OK',
        1625075289
    );

-- Table for database 7010 testdb_staging 
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7010,
        'tbl1',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        12234,
        65536,
        16384,
        'OK',
        1624475289
    );

-- Table for database 7014 testdb_prod
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7014,
        'tbl1',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1234,
        32768,
        4096,
        'OK',
        1624475289
    );

-- Table for database 7003 shop
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7003,
        'product',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1000,
        4096,
        0,
        'OK',
        1624275289
    );

-- Table for database 7007 shop
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7007,
        'product',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1000,
        4096,
        1024,
        'OK',
        1624575289
    );

-- Table for database 7004 blog
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7004,
        'user',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1000,
        4096,
        1024,
        'OK',
        1624275289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7004,
        'post',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        1000,
        8192,
        1024,
        'OK',
        1624275289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7004,
        'comment',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        30000,
        65536,
        4096,
        'OK',
        1624275289
    );

-- Table for database 7008 blog
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7008,
        'user',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        100,
        4096,
        0,
        'OK',
        1624675289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7008,
        'post',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        300,
        4096,
        1024,
        'OK',
        1624675289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7008,
        'comment',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        2000,
        65536,
        2048,
        'OK',
        1624675289
    );

-- Table for database 7012 blog
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7012,
        'user',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        200,
        4096,
        1024,
        'OK',
        1622675289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7012,
        'post',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        500,
        4096,
        1024,
        'OK',
        1622675289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7012,
        'comment',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        4000,
        65536,
        65536,
        'OK',
        1622675289
    );

-- Table for database 7016 blog
INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7016,
        'user',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        200,
        4096,
        1024,
        'OK',
        1622275289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7016,
        'post',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        500,
        4096,
        1024,
        'OK',
        1622275289
    );

INSERT INTO
    tbl (
        creator_id,
        updater_id,
        database_id,
        name,
        ENGINE,
        `collation`,
        row_count,
        data_size,
        index_size,
        sync_status,
        last_successful_sync_ts
    )
VALUES
    (
        101,
        101,
        7016,
        'comment',
        'InnoDB',
        'utf8mb4_0900_ai_ci',
        7000,
        8192,
        0,
        'OK',
        1622275289
    );