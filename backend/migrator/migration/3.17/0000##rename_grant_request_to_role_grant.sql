UPDATE issue
SET type = 'ROLE_GRANT'
WHERE type = 'GRANT_REQUEST';

UPDATE issue
SET payload = (payload - 'grantRequest') || jsonb_build_object('roleGrant', payload->'grantRequest')
WHERE payload ? 'grantRequest';
