ALTER TABLE principal DROP CONSTRAINT principal_type_check;
ALTER TABLE principal ADD CONSTRAINT principal_type_check CHECK (type IN ('END_USER', 'SYSTEM_BOT', 'SERVICE_ACCOUNT'));
