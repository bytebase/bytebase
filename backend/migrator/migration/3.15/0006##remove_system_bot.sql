-- Make task_run.creator nullable (for system-generated task runs)
ALTER TABLE task_run ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE task_run DROP CONSTRAINT task_run_creator_fkey;
ALTER TABLE task_run ADD CONSTRAINT task_run_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Convert existing SystemBot task runs to NULL
UPDATE task_run SET creator = NULL
  WHERE creator = 'support@bytebase.com';

-- Backfill tables that have support@bytebase.com from GitOps workflows
-- Reassign to the first active user (or fail if none exists)
DO $$
DECLARE
  fallback_user TEXT;
BEGIN
  -- Find the first active END_USER (by lowest ID, typically the initial setup user)
  SELECT email INTO fallback_user
  FROM principal
  WHERE email != 'support@bytebase.com'
    AND type = 'END_USER'
    AND NOT deleted
  ORDER BY id
  LIMIT 1;

  -- If no user found, raise error
  IF fallback_user IS NULL THEN
    RAISE EXCEPTION 'No active user found to reassign SystemBot-created records. Please create a user first.';
  END IF;

  -- Reassign all support@bytebase.com creators to the fallback user
  -- These tables must have real users, not system bot
  UPDATE sheet SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE plan SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE issue SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE issue_comment SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE release SET creator = fallback_user WHERE creator = 'support@bytebase.com';
END $$;

-- Delete all SYSTEM_BOT principal rows (support@bytebase.com and allUsers)
DELETE FROM principal WHERE type = 'SYSTEM_BOT';

-- Remove SYSTEM_BOT from principal type constraint
ALTER TABLE principal DROP CONSTRAINT principal_type_check;
ALTER TABLE principal ADD CONSTRAINT principal_type_check
  CHECK (type IN ('END_USER', 'SERVICE_ACCOUNT', 'WORKLOAD_IDENTITY'));

-- Update principal_project_type_check to remove SYSTEM_BOT
ALTER TABLE principal DROP CONSTRAINT principal_project_type_check;
ALTER TABLE principal ADD CONSTRAINT principal_project_type_check CHECK (
  (type = 'END_USER' AND project IS NULL) OR
  (type IN ('SERVICE_ACCOUNT', 'WORKLOAD_IDENTITY'))
);
