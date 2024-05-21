CREATE TABLE user_group (
    id SERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    email TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX idx_user_group_unique_resource_id ON user_group(resource_id);

CREATE UNIQUE INDEX idx_user_group_unique_email ON user_group(email);

ALTER SEQUENCE user_group_id_seq RESTART WITH 101;
