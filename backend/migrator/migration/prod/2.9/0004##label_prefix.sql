UPDATE db SET
    metadata = metadata || jsonb_build_object('labels', (metadata->'labels')::JSONB - 'bb.environment')
WHERE metadata ? 'labels' AND (metadata->'labels')::JSONB ? 'bb.environment';

UPDATE db SET
    metadata = metadata || jsonb_build_object('labels', REPLACE((metadata->'labels')::TEXT, 'bb.', '')::JSONB)
WHERE metadata ? 'labels';

UPDATE deployment_config SET config = REPLACE(config::TEXT, 'bb.', '')::JSONB;