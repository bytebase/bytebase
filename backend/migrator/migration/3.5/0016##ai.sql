INSERT INTO setting (name, value) VALUES ('bb.ai',
    json_build_object(
        'enabled', true,
        'provider', 'OPEN_AI',
        'apiKey', (SELECT COALESCE(value, '') FROM setting WHERE name = 'bb.plugin.openai.key'),
        'endpoint', (
            SELECT TRIM(TRAILING '/' FROM (CASE WHEN COALESCE(value, '') = '' THEN 'https://api.openai.com/' ELSE COALESCE(value, '') END)) || '/v1/chat/completions'
            FROM setting WHERE name = 'bb.plugin.openai.endpoint'),
        'model', 'gpt-3.5-turbo'
    )
);
DELETE FROM setting WHERE name = 'bb.ai' AND (value::jsonb)->>'apiKey' = '';
DELETE FROM setting WHERE name LIKE 'bb.plugin.openai%';