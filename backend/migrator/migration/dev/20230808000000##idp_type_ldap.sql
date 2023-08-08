ALTER TABLE idp DROP CONSTRAINT idp_type_check;
ALTER TABLE idp ADD CONSTRAINT idp_type_check CHECK (type IN ('OAUTH2', 'OIDC', 'LDAP'));
