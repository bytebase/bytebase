UPDATE instance SET metadata = REPLACE(metadata::text, 'sshObfuscatedPassword', 'obfuscatedSshPassword')::jsonb;
UPDATE instance SET metadata = REPLACE(metadata::text, 'sshObfuscatedPrivateKey', 'obfuscatedSshPrivateKey')::jsonb;
UPDATE instance SET metadata = REPLACE(metadata::text, 'authenticationPrivateKeyObfuscated', 'obfuscatedAuthenticationPrivateKey')::jsonb;
UPDATE instance SET metadata = REPLACE(metadata::text, 'masterObfuscatedPassword', 'obfuscatedMasterPassword')::jsonb;