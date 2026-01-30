-- Make issue_comment.creator nullable
ALTER TABLE issue_comment ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE issue_comment DROP CONSTRAINT issue_comment_creator_fkey;
ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Make task_run.creator nullable
ALTER TABLE task_run ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE task_run DROP CONSTRAINT task_run_creator_fkey;
ALTER TABLE task_run ADD CONSTRAINT task_run_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Convert all existing SystemBot records to NULL
UPDATE issue_comment SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE task_run SET creator = NULL
  WHERE creator = 'support@bytebase.com';

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
