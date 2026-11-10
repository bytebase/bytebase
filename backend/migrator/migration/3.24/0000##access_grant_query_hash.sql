UPDATE access_grant
SET payload = jsonb_set(
    payload,
    '{queryHash}',
    to_jsonb(encode(sha256(convert_to(btrim(payload->>'query', E' \t\n\r\x0b\f'), 'UTF8')), 'hex'))
)
WHERE payload->>'query' IS NOT NULL;

CREATE INDEX idx_access_grant_project_creator_query_hash ON access_grant(project, creator, (payload->>'queryHash'));
