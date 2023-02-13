INSERT INTO idp (resource_id, name, domain, type, config)
SELECT
  CONCAT(
    'idp-',
    LEFT(uuid_in(md5(random()::text || random()::text)::cstring)::TEXT, 8)
  ) AS resource_id,
  name AS name,
  TRIM(
    LEADING 'http://' FROM
      TRIM(
        LEADING 'https://' FROM instance_url
      )
  ) AS domain,
  'OAUTH2' AS type,
  jsonb_build_object(
    'authUrl', instance_url || '/oauth/authorize',
    'tokenUrl', instance_url || '/oauth/token',
    'userInfoUrl', instance_url || '/api/v4/user',
    'clientId', application_id,
    'clientSecret', secret,
    'scopes', jsonb_build_array('api'),
    'fieldMapping', '{"email": "email", "identifier": "email", "displayName": "name"}'::jsonb
  ) AS config
FROM vcs
WHERE type = 'GITLAB_SELF_HOST';