-- Tracks one sample setup lifecycle per workspace. The payload is owned by the
-- selected sample manager implementation.
CREATE TABLE sample_instance_setup (
    workspace text PRIMARY KEY REFERENCES workspace(resource_id),
    replica_id text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    activated_at timestamptz,
    expires_at timestamptz,
    deleted_at timestamptz,
    CHECK (expires_at IS NULL OR activated_at IS NOT NULL),
    CHECK (deleted_at IS NULL OR activated_at IS NOT NULL)
);

INSERT INTO sample_instance_setup (
    workspace,
    replica_id,
    payload,
    created_at,
    updated_at,
    activated_at,
    expires_at,
    deleted_at
)
SELECT
    workspace,
    replica_id,
    jsonb_build_object(
        'projectId', project,
        'instanceId', instance,
        'title', 'Sample Project Instance',
        'databaseName', db_name,
        'roleName', role_name
    ),
    created_at,
    created_at,
    CASE WHEN expires_at IS NULL THEN NULL ELSE expires_at - interval '7 days' END,
    expires_at,
    deleted_at
FROM sample_project_instance;

DO $$
DECLARE
    old_count bigint;
    setup_count bigint;
BEGIN
    SELECT COUNT(*) INTO old_count FROM sample_project_instance;
    SELECT COUNT(*) INTO setup_count FROM sample_instance_setup;
    IF old_count <> setup_count THEN
        RAISE EXCEPTION 'sample Project Instance migration count mismatch: old %, setup %', old_count, setup_count;
    END IF;
END $$;

DROP TABLE sample_project_instance;
