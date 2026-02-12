CREATE TABLE access_grant (
    id text PRIMARY KEY,
    project text NOT NULL REFERENCES project(resource_id),
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    status text NOT NULL DEFAULT 'PENDING',
    expire_time timestamptz,
    -- Stored as AccessGrantPayload (proto/store/store/access_grant.proto)
    payload jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_grant_project_creator_expire_time ON access_grant(project, creator, expire_time);
