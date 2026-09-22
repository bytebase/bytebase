-- A review result is posted by a reviewer, not a person: its creator is NULL
-- and review_metadata names the reviewer, as task_run.creator is NULL for
-- system-generated runs.
ALTER TABLE issue_comment ALTER COLUMN creator DROP NOT NULL;
