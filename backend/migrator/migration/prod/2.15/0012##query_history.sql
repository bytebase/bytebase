CREATE TABLE query_history (
    id BIGSERIAL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    project_id TEXT NOT NULL, -- the project resource id
    database TEXT NOT NULL, -- the database resource name, for example, instances/{instance}/databases/{database}
    statement TEXT NOT NULL,
    type TEXT NOT NULL, -- the history type, support QUERY and EXPORT.
    payload JSONB NOT NULL DEFAULT '{}' -- saved for details, like error, duration, etc.
);

CREATE INDEX idx_query_history_creator_id_created_ts_project_id ON query_history(creator_id, created_ts, project_id DESC);

ALTER SEQUENCE query_history_id_seq RESTART WITH 101;

INSERT INTO query_history(creator_id, created_ts, project_id, database, statement, type)
SELECT
    activity.creator_id,
    activity.created_ts,
    REPLACE(activity.resource_container, 'projects/', ''),
    (SELECT 'instances/' || instance.resource_id || '/databases/' || db.name FROM instance JOIN db ON instance.id = db.instance_id WHERE (db.id)::text = (activity.payload->'databaseId')::text),
    TRIM('"' FROM (activity.payload->'statement')::text),
    'QUERY'
FROM activity
JOIN db ON (db.id)::text = (activity.payload->'databaseId')::text
WHERE activity.type = 'bb.sql.query'