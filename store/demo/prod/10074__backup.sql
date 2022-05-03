-- Backup for database 7015 (shop database in prod environment)
INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7401,
        101,
        1629136800,
        101,
        1629136800,
        7015,
        'shop-prod-20210817T094000',
        'DONE',
        'MANUAL',
        'LOCAL',
        'v1_20210817094000',
        'data/backup/db/7015/shop-prod-20210817T094000.sql',
        ''
    );

INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7402,
        101,
        1629250800,
        101,
        1629250800,
        7015,
        'shop-prod-20210818T140000',
        'DONE',
        'MANUAL',
        'LOCAL',
        'v2_20210818140000',
        'data/backup/db/7015/shop-prod-20210818T140000.sql',
        ''
    );

INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7403,
        101,
        1627754400,
        101,
        1627754400,
        7015,
        'shop-prod-20210801T020000-autobackup',
        'FAILED',
        'AUTOMATIC',
        'LOCAL',
        'v3_20210801020000',
        'data/backup/db/7015/shop-prod-20210801T020000-autobackup.sql',
        'Something unfortunate happened. Here is a long long long long long long long long long long long long long long long long long long long long long long long long error message.'
    );

INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7404,
        101,
        1628359200,
        101,
        1628359200,
        7015,
        'shop-prod-20210808T020000-autobackup',
        'DONE',
        'AUTOMATIC',
        'LOCAL',
        'v3_20210808020000',
        'data/backup/db/7015/shop-prod-20210808T020000-autobackup.sql',
        ''
    );

-- Backup for database 7016 (blog database in prod environment)
INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7405,
        101,
        1629250800,
        101,
        1629250800,
        7016,
        'blog-prod-20210818T094000',
        'DONE',
        'MANUAL',
        'LOCAL',
        'v1_20210818094000',
        'data/backup/db/7016/blog-prod-20210818T094000.sql',
        ''
    );

INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7406,
        101,
        1629337200,
        101,
        1629337200,
        7016,
        'blog-prod-20210819T140000',
        'DONE',
        'MANUAL',
        'LOCAL',
        'v2_20210819140000',
        'data/backup/db/7016/blog-prod-20210819T140000.sql',
        ''
    );

INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7407,
        101,
        1627754400,
        101,
        1627754400,
        7016,
        'blog-prod-20210801T020000-autobackup',
        'FAILED',
        'AUTOMATIC',
        'LOCAL',
        'v3_20210801020000',
        'data/backup/db/7016/blog-prod-20210801T020000-autobackup.sql',
        'Something unfortunate happened. Here is a long long long long long long long long long long long long long long long long long long long long long long long long error message.'
    );

INSERT INTO
    backup (
        id,
        creator_id,
        created_ts,
        updater_id,
        updated_ts,
        database_id,
        name,
        status,
        type,
        storage_backend,
        migration_history_version,
        path,
        comment
    )
VALUES
    (
        7408,
        101,
        1628359200,
        101,
        1628359200,
        7016,
        'blog-prod-20210808T020000-autobackup',
        'DONE',
        'AUTOMATIC',
        'LOCAL',
        'v3_20210808020000',
        'data/backup/db/7016/blog-prod-20210808T020000-autobackup.sql',
        ''
    );

ALTER SEQUENCE backup_id_seq RESTART WITH 7409;
