-- Migration to replace integer IDs with email strings for principal references

-- 1. sheet
ALTER TABLE sheet ADD COLUMN creator text;
UPDATE sheet SET creator = p.email FROM principal p WHERE sheet.creator_id = p.id;
ALTER TABLE sheet ALTER COLUMN creator SET NOT NULL;
ALTER TABLE sheet DROP COLUMN creator_id;
ALTER TABLE sheet ADD FOREIGN KEY (creator) REFERENCES principal(email);

-- 2. pipeline
ALTER TABLE pipeline ADD COLUMN creator text;
UPDATE pipeline SET creator = p.email FROM principal p WHERE pipeline.creator_id = p.id;
ALTER TABLE pipeline ALTER COLUMN creator SET NOT NULL;
ALTER TABLE pipeline DROP COLUMN creator_id;
ALTER TABLE pipeline ADD FOREIGN KEY (creator) REFERENCES principal(email);

-- 3. task_run
ALTER TABLE task_run ADD COLUMN creator text;
UPDATE task_run SET creator = p.email FROM principal p WHERE task_run.creator_id = p.id;
ALTER TABLE task_run ALTER COLUMN creator SET NOT NULL;
ALTER TABLE task_run DROP COLUMN creator_id;
ALTER TABLE task_run ADD FOREIGN KEY (creator) REFERENCES principal(email);
-- Recreate indexes/constraints if needed (none used creator_id)

-- 4. plan
ALTER TABLE plan ADD COLUMN creator text;
UPDATE plan SET creator = p.email FROM principal p WHERE plan.creator_id = p.id;
ALTER TABLE plan ALTER COLUMN creator SET NOT NULL;
ALTER TABLE plan DROP COLUMN creator_id;
ALTER TABLE plan ADD FOREIGN KEY (creator) REFERENCES principal(email);

-- 5. issue
ALTER TABLE issue ADD COLUMN creator text;
UPDATE issue SET creator = p.email FROM principal p WHERE issue.creator_id = p.id;
ALTER TABLE issue ALTER COLUMN creator SET NOT NULL;
ALTER TABLE issue DROP COLUMN creator_id;
ALTER TABLE issue ADD FOREIGN KEY (creator) REFERENCES principal(email);
-- Recreate index
DROP INDEX IF EXISTS idx_issue_creator_id;
CREATE INDEX idx_issue_creator ON issue(creator);

-- 6. issue_comment
ALTER TABLE issue_comment ADD COLUMN creator text;
UPDATE issue_comment SET creator = p.email FROM principal p WHERE issue_comment.creator_id = p.id;
ALTER TABLE issue_comment ALTER COLUMN creator SET NOT NULL;
ALTER TABLE issue_comment DROP COLUMN creator_id;
ALTER TABLE issue_comment ADD FOREIGN KEY (creator) REFERENCES principal(email);

-- 7. query_history
ALTER TABLE query_history ADD COLUMN creator text;
UPDATE query_history SET creator = p.email FROM principal p WHERE query_history.creator_id = p.id;
ALTER TABLE query_history ALTER COLUMN creator SET NOT NULL;
ALTER TABLE query_history DROP COLUMN creator_id;
ALTER TABLE query_history ADD FOREIGN KEY (creator) REFERENCES principal(email);
-- Recreate index
DROP INDEX IF EXISTS idx_query_history_creator_id_created_at_project_id;
CREATE INDEX idx_query_history_creator_created_at_project_id ON query_history(creator, created_at, project_id DESC);

-- 8. worksheet
ALTER TABLE worksheet ADD COLUMN creator text;
UPDATE worksheet SET creator = p.email FROM principal p WHERE worksheet.creator_id = p.id;
ALTER TABLE worksheet ALTER COLUMN creator SET NOT NULL;
ALTER TABLE worksheet DROP COLUMN creator_id;
ALTER TABLE worksheet ADD FOREIGN KEY (creator) REFERENCES principal(email);
-- Recreate index
DROP INDEX IF EXISTS idx_worksheet_creator_id_project;
CREATE INDEX idx_worksheet_creator_project ON worksheet(creator, project);

-- 9. worksheet_organizer
ALTER TABLE worksheet_organizer ADD COLUMN principal text;
UPDATE worksheet_organizer SET principal = p.email FROM principal p WHERE worksheet_organizer.principal_id = p.id;
ALTER TABLE worksheet_organizer ALTER COLUMN principal SET NOT NULL;
ALTER TABLE worksheet_organizer DROP COLUMN principal_id;
ALTER TABLE worksheet_organizer ADD FOREIGN KEY (principal) REFERENCES principal(email);
-- Recreate indexes
DROP INDEX IF EXISTS idx_worksheet_organizer_unique_sheet_id_principal_id;
CREATE UNIQUE INDEX idx_worksheet_organizer_unique_sheet_id_principal ON worksheet_organizer(worksheet_id, principal);
DROP INDEX IF EXISTS idx_worksheet_organizer_principal_id;
CREATE INDEX idx_worksheet_organizer_principal ON worksheet_organizer(principal);

-- 10. revision
ALTER TABLE revision ADD COLUMN deleter text;
UPDATE revision SET deleter = p.email FROM principal p WHERE revision.deleter_id = p.id;
-- deleter is nullable
ALTER TABLE revision DROP COLUMN deleter_id;
ALTER TABLE revision ADD FOREIGN KEY (deleter) REFERENCES principal(email);

-- 11. release
ALTER TABLE release ADD COLUMN creator text;
UPDATE release SET creator = p.email FROM principal p WHERE release.creator_id = p.id;
ALTER TABLE release ALTER COLUMN creator SET NOT NULL;
ALTER TABLE release DROP COLUMN creator_id;
ALTER TABLE release ADD FOREIGN KEY (creator) REFERENCES principal(email);
