UPDATE db SET
    metadata = metadata || jsonb_build_object('labels', (metadata->'labels')::JSONB - 'bb.environment')
WHERE metadata ? 'labels';

UPDATE db SET
    metadata = metadata || jsonb_build_object('labels', REPLACE((metadata->'labels')::TEXT, 'bb.', '')::JSONB)
WHERE metadata ? 'labels';

select (metadata->'labels')::JSONB - 'tenant1' from db WHERE (metadata->'labels')::JSONB ? 'tenant11';

UPDATE deployment_config SET config = REPLACE(config::TEXT, 'bb.', '')::JSONB;