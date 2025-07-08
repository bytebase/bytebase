-- Convert LDAP security protocol from string to enum string values
-- The mapping is:
-- "starttls" -> "START_TLS"
-- "ldaps" -> "LDAPS"
-- "" (empty string) or NULL -> remove the field (will default to SECURITY_PROTOCOL_UNSPECIFIED)
--
-- Note: For LDAP type, the config column stores the LDAP configuration directly,
-- not wrapped in a "ldapConfig" field. Protojson uses enum string names, not numbers.

-- First, update the non-empty values
UPDATE idp
SET config = jsonb_set(
    config,
    '{securityProtocol}',
    CASE
        WHEN config->>'securityProtocol' = 'starttls' THEN '"START_TLS"'::jsonb
        WHEN config->>'securityProtocol' = 'ldaps' THEN '"LDAPS"'::jsonb
    END
)
WHERE type = 'LDAP'
  AND config->>'securityProtocol' IN ('starttls', 'ldaps');

-- Then, remove the field for empty or null values
UPDATE idp
SET config = config - 'securityProtocol'
WHERE type = 'LDAP'
  AND (config->>'securityProtocol' = '' OR config->>'securityProtocol' IS NULL);
