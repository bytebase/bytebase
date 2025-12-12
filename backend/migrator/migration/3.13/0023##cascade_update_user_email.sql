-- Add ON UPDATE CASCADE to user email foreign keys

-- 1. sheet
ALTER TABLE sheet DROP CONSTRAINT IF EXISTS sheet_creator_fkey;
ALTER TABLE sheet ADD CONSTRAINT sheet_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 2. pipeline
ALTER TABLE pipeline DROP CONSTRAINT IF EXISTS pipeline_creator_fkey;
ALTER TABLE pipeline ADD CONSTRAINT pipeline_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 3. task_run
ALTER TABLE task_run DROP CONSTRAINT IF EXISTS task_run_creator_fkey;
ALTER TABLE task_run ADD CONSTRAINT task_run_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 4. plan
ALTER TABLE plan DROP CONSTRAINT IF EXISTS plan_creator_fkey;
ALTER TABLE plan ADD CONSTRAINT plan_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 5. issue
ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_creator_fkey;
ALTER TABLE issue ADD CONSTRAINT issue_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 6. issue_comment
ALTER TABLE issue_comment DROP CONSTRAINT IF EXISTS issue_comment_creator_fkey;
ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 7. query_history
ALTER TABLE query_history DROP CONSTRAINT IF EXISTS query_history_creator_fkey;
ALTER TABLE query_history ADD CONSTRAINT query_history_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 8. worksheet
ALTER TABLE worksheet DROP CONSTRAINT IF EXISTS worksheet_creator_fkey;
ALTER TABLE worksheet ADD CONSTRAINT worksheet_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;

-- 9. worksheet_organizer
ALTER TABLE worksheet_organizer DROP CONSTRAINT IF EXISTS worksheet_organizer_principal_fkey;
ALTER TABLE worksheet_organizer ADD CONSTRAINT worksheet_organizer_principal_fkey FOREIGN KEY (principal) REFERENCES principal(email) ON UPDATE CASCADE;

-- 10. revision
ALTER TABLE revision DROP CONSTRAINT IF EXISTS revision_deleter_fkey;
ALTER TABLE revision ADD CONSTRAINT revision_deleter_fkey FOREIGN KEY (deleter) REFERENCES principal(email) ON UPDATE CASCADE;

-- 11. release
ALTER TABLE release DROP CONSTRAINT IF EXISTS release_creator_fkey;
ALTER TABLE release ADD CONSTRAINT release_creator_fkey FOREIGN KEY (creator) REFERENCES principal(email) ON UPDATE CASCADE;
