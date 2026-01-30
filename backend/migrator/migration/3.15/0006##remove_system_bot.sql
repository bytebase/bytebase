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

-- Make issue.creator nullable
ALTER TABLE issue ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE issue DROP CONSTRAINT issue_creator_fkey;
ALTER TABLE issue ADD CONSTRAINT issue_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Make plan.creator nullable
ALTER TABLE plan ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE plan DROP CONSTRAINT plan_creator_fkey;
ALTER TABLE plan ADD CONSTRAINT plan_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Make release.creator nullable
ALTER TABLE release ALTER COLUMN creator DROP NOT NULL;
ALTER TABLE release DROP CONSTRAINT release_creator_fkey;
ALTER TABLE release ADD CONSTRAINT release_creator_fkey
  FOREIGN KEY (creator) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Update revision.deleter to cascade delete (already nullable)
ALTER TABLE revision DROP CONSTRAINT revision_deleter_fkey;
ALTER TABLE revision ADD CONSTRAINT revision_deleter_fkey
  FOREIGN KEY (deleter) REFERENCES principal(email)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Convert all existing SystemBot records to NULL
UPDATE issue_comment SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE task_run SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE issue SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE plan SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE release SET creator = NULL
  WHERE creator = 'support@bytebase.com';

UPDATE revision SET deleter = NULL
  WHERE deleter = 'support@bytebase.com';

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
