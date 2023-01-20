-- Database anomalies
INSERT INTO
    anomaly (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        type,
        payload
    )
VALUES
    (
        19001,
        1,
        1,
        6004,
        7014,
        'bb.anomaly.database.connection',
        '{"detail":"failed to connect database at mysql.prod.example.com:3306 with user \"root\": dial tcp: lookup mysql.prod.example.com: no such host"}'
    );

INSERT INTO
    anomaly (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        type,
        payload
    )
VALUES
    (
        19002,
        1,
        1,
        6004,
        7014,
        'bb.anomaly.database.backup.policy-violation',
        '{"environmentId":5004,"expectedSchedule":"DAILY","actualSchedule":"UNSET"}'
    );

INSERT INTO
    anomaly (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        type,
        payload
    )
VALUES
    (
        19003,
        1,
        1,
        6003,
        7012,
        'bb.anomaly.database.connection',
        '{"detail":"failed to connect database at mysql.staging.example.com:3306 with user \"root\": dial tcp: lookup mysql.staging.example.com: no such host"}'
    );

INSERT INTO
    anomaly (
        id,
        creator_id,
        updater_id,
        instance_id,
        database_id,
        type,
        payload
    )
VALUES
    (
        19004,
        1,
        1,
        6003,
        7012,
        'bb.anomaly.database.backup.policy-violation',
        '{"environmentId":5003,"expectedSchedule":"WEEKLY","actualSchedule":"UNSET"}'
    );

-- Instance anomalies
INSERT INTO
    anomaly (
        id,
        creator_id,
        updater_id,
        instance_id,
        type,
        payload
    )
VALUES
    (
        19005,
        1,
        1,
        6004,
        'bb.anomaly.instance.connection',
        '{"detail":"failed to connect database at mysql.prod.example.com:3306 with user \"root\": dial tcp: lookup mysql.prod.example.com: no such host"}'
    );

INSERT INTO
    anomaly (
        id,
        creator_id,
        updater_id,
        instance_id,
        type,
        payload
    )
VALUES
    (
        19006,
        1,
        1,
        6003,
        'bb.anomaly.instance.connection',
        '{"detail":"failed to connect database at mysql.staging.example.com:3306 with user \"admin\": dial tcp: lookup mysql.staging.example.com: no such host"}'
    );

ALTER SEQUENCE anomaly_id_seq RESTART WITH 19007;
