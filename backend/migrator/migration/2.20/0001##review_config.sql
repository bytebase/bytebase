-- Create review config table.
CREATE TABLE review_config
(
    id TEXT NOT NULL PRIMARY KEY,
    row_status row_status NOT NULL DEFAULT 'NORMAL',
    creator_id INTEGER NOT NULL REFERENCES principal (id),
    created_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    updater_id INTEGER NOT NULL REFERENCES principal (id),
    updated_ts BIGINT NOT NULL DEFAULT extract(epoch from now()),
    name TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'
);

-- Migrate sql review policy to the new table.
INSERT INTO review_config
    (id, row_status, creator_id, created_ts, updater_id, updated_ts, name, payload)
SELECT
    environment.resource_id,
    policy.row_status,
    policy.creator_id,
    policy.created_ts,
    policy.updater_id,
    policy.updated_ts,
    policy.payload->>'name',
    jsonb_build_object('sqlReviewRules', policy.payload->'ruleList')
FROM policy
INNER JOIN environment ON policy.resource_id = environment.id
WHERE type = 'bb.policy.sql-review';

-- Migrate environment sql review policy to bb.policy.tag policy.
INSERT INTO policy
    (creator_id, created_ts, updater_id, updated_ts, resource_type, resource_id, type, payload)
SELECT
    policy.creator_id,
    policy.created_ts,
    policy.updater_id,
    policy.updated_ts,
    policy.resource_type,
    policy.resource_id,
    'bb.policy.tag',
    -- We're using environment.resource_id as review config resource_id.
    jsonb_build_object('tags', jsonb_build_object('bb.tag.review_config', CONCAT('reviewConfigs/', environment.resource_id)))
FROM policy
INNER JOIN environment ON policy.resource_id = environment.id
WHERE type = 'bb.policy.sql-review';