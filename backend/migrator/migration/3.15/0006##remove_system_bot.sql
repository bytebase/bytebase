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
-- Reassign to the first workspace owner (or fail if none exists)
DO $$
DECLARE
  fallback_user TEXT;
BEGIN
  -- Find the first active workspace owner
  SELECT email INTO fallback_user
  FROM principal p
  JOIN member m ON p.email = m.principal
  WHERE p.email != 'support@bytebase.com'
    AND m.role = 'roles/workspaceAdmin'
    AND NOT p.deleted
  ORDER BY p.id
  LIMIT 1;

  -- If no workspace admin found, try to find any active END_USER
  IF fallback_user IS NULL THEN
    SELECT email INTO fallback_user
    FROM principal
    WHERE email != 'support@bytebase.com'
      AND type = 'END_USER'
      AND NOT deleted
    ORDER BY id
    LIMIT 1;
  END IF;

  -- If still no user found, raise error
  IF fallback_user IS NULL THEN
    RAISE EXCEPTION 'No active user found to reassign SystemBot-created records. Please create a user first.';
  END IF;

  -- Reassign all support@bytebase.com creators to the fallback user
  -- These tables must have real users, not system bot
  UPDATE sheet SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE plan SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE issue SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE issue_comment SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE query_history SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE worksheet SET creator = fallback_user WHERE creator = 'support@bytebase.com';
  UPDATE release SET creator = fallback_user WHERE creator = 'support@bytebase.com';
END $$;

-- Clean up other tables that might reference SystemBot
-- Set revision.deleter to NULL for SystemBot-deleted revisions
UPDATE revision SET deleter = NULL
  WHERE deleter = 'support@bytebase.com';

-- Remove worksheet organizer entries for SystemBot
DELETE FROM worksheet_organizer
  WHERE principal = 'support@bytebase.com';

-- Remove oauth2 tokens for SystemBot (if any)
DELETE FROM oauth2_authorization_code
  WHERE user_email = 'support@bytebase.com';
DELETE FROM oauth2_refresh_token
  WHERE user_email = 'support@bytebase.com';
DELETE FROM web_refresh_token
  WHERE user_email = 'support@bytebase.com';

-- Delete the SystemBot principal row
DELETE FROM principal WHERE id = 1 AND email = 'support@bytebase.com';

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
