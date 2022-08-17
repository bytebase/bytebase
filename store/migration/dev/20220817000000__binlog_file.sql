-- binlog_file stores metadata for MySQL binlog files.
CREATE TABLE binlog_file (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),

    instance_id INTEGER NOT NULL REFERENCES instance (id),
    storage_backend TEXT NOT NULL CHECK (storage_backend IN ('LOCAL', 'S3', 'GCS')),
    path TEXT NOT NULL,
    -- The first binlog event's timestamp in this file.
    -- This can be used to find the binlog file containing a given timestamp quickly.
    first_event_ts BIGINT NOT NULL,
    -- The GTIDs in this binlog file.
    -- This can be used to find the binlog file containing a given timestamp quickly if GTID is enabled for the MySQL server.
    -- See https://dev.mysql.com/doc/refman/8.0/en/replication-gtids-concepts.html#replication-gtids-concepts-gtid-sets for detailed format.
    -- An example: "2174B383-5441-11E8-B90A-C80AA9429562:1-3, 24DA167-0C0C-11E8-8442-00059A3C7B00:1-19"
    gtid_set TEXT
);

CREATE INDEX idx_binlog_file_instance_id ON binlog_file(instance_id);

ALTER SEQUENCE binlog_file_id_seq RESTART WITH 101;

CREATE TRIGGER update_binlog_file_updated_ts
BEFORE
UPDATE
    ON binlog_file FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
