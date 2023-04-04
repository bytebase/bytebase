-- slow_query stores slow query statistics for each database.
CREATE TABLE slow_query (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    instance_id INTEGER NOT NULL REFERENCES instance (id),
    database_id INTEGER NOT NULL REFERENCES db (id),
    log_date INTEGER NOT NULL,
    sync_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    slow_query_statistics JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_slow_query_database_id_log_date ON slow_query (database_id, log_date);

ALTER SEQUENCE slow_query_id_seq RESTART WITH 101;

CREATE TRIGGER update_slow_query_updated_ts
BEFORE
UPDATE
    ON slow_query FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();
