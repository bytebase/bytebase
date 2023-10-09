UPDATE db SET
    metadata = metadata - 'payload' || jsonb_build_object('labels', REPLACE((metadata->'labels')::TEXT, 'bb.', '')::JSONB)
WHERE metadata ? 'labels';

UPDATE deployment_config SET config = REPLACE(config::TEXT, 'bb.', '')::JSONB;