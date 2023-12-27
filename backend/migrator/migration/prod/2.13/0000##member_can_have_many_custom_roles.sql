ALTER TABLE member DROP CONSTRAINT IF EXISTS member_role_check;

DROP INDEX IF EXISTS idx_member_unique_principal_id;

CREATE INDEX IF NOT EXISTS idx_member_principal_id ON member (principal_id);

ALTER TABLE member DROP COLUMN IF EXISTS status;
