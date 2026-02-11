CREATE TABLE access_grant (
    id bigserial PRIMARY KEY,
    project text NOT NULL REFERENCES project(resource_id),
    creator text NOT NULL REFERENCES principal(email) ON UPDATE CASCADE,
    status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'ACTIVE', 'REVOKED')),
    expire_time timestamptz NOT NULL,
    issue_id bigint REFERENCES issue(id),
    targets jsonb NOT NULL DEFAULT '[]',
    query text NOT NULL DEFAULT '',
    unmask boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_access_grant_project_creator_expire_time ON access_grant(project, creator, expire_time);

ALTER SEQUENCE access_grant_id_seq RESTART WITH 101;
