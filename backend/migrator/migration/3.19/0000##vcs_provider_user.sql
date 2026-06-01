CREATE TABLE vcs_provider_user (
    workspace text NOT NULL REFERENCES workspace(resource_id),
    vcs_type text NOT NULL,
    user_id text NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    payload jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (workspace, vcs_type, user_id)
);

CREATE INDEX idx_vcs_provider_user_workspace_last_seen_at
    ON vcs_provider_user(workspace, last_seen_at DESC);

CREATE INDEX idx_vcs_provider_user_last_seen_at
    ON vcs_provider_user(last_seen_at);
