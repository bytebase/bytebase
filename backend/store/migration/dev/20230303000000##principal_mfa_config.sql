ALTER TABLE principal ADD mfa_config JSONB NOT NULL DEFAULT '{}';

ALTER TABLE principal ADD recovery_codes TEXT ARRAY NOT NULL DEFAULT '{}';
