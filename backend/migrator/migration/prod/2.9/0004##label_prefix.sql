UPDATE db SET
    metadata = metadata - 'payload' || jsonb_build_object('labels', REPLACE((metadata->'labels')::TEXT, 'tenant', 'tenant1')::JSONB)
WHERE metadata ? 'labels';
