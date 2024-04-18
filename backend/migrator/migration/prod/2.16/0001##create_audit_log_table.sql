CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    payload JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_audit_log_created_ts ON audit_log(created_ts);

ALTER SEQUENCE audit_log_id_seq RESTART WITH 101;
