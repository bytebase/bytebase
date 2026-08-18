-- Tracks the lifetime sample Project Instance entitlement independently of
-- Bytebase metadata and its physical PostgreSQL resources.
CREATE TABLE sample_project_instance (
    workspace text PRIMARY KEY,
    project text NOT NULL,
    instance text NOT NULL UNIQUE,
    db_name text NOT NULL UNIQUE,
    role_name text NOT NULL UNIQUE,
    ownership_known boolean NOT NULL DEFAULT FALSE,
    database_created boolean NOT NULL DEFAULT FALSE,
    role_created boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    deleted_at timestamptz,
    CHECK (deleted_at IS NULL OR expires_at IS NOT NULL)
);
