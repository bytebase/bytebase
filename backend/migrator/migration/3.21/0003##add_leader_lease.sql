CREATE TABLE leader_lease (
    type TEXT PRIMARY KEY,
    replica_id TEXT NOT NULL,
    generation BIGINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
