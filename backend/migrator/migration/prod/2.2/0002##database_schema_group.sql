CREATE TABLE db_group (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    placeholder TEXT NOT NULL DEFAULT '',
    expression JSONB NOT NULL DEFAULT '{}'
);

ALTER SEQUENCE db_group_id_seq RESTART WITH 101;

CREATE TRIGGER update_db_group_updated_ts
BEFORE
UPDATE
    ON db_group FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();

CREATE TABLE schema_group (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    db_group_id INTEGER NOT NULL REFERENCES db_group (id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    placeholder TEXT NOT NULL DEFAULT '',
    expression JSONB NOT NULL DEFAULT '{}'
);

ALTER SEQUENCE schema_group_id_seq RESTART WITH 101;

CREATE TRIGGER update_schema_group_updated_ts
BEFORE
UPDATE
    ON schema_group FOR EACH ROW
EXECUTE FUNCTION trigger_update_updated_ts();