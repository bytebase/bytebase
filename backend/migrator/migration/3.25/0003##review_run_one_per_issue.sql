-- An issue has one review run (BYT-10505). Each reviewer used to have a row of
-- its own, told apart by type. Rows of those types are no longer read, so they
-- go, and the type leaves the key.
DELETE FROM review_run WHERE type <> 'REVIEW';
ALTER TABLE review_run DROP CONSTRAINT review_run_pkey;
ALTER TABLE review_run DROP COLUMN type;
ALTER TABLE review_run ADD PRIMARY KEY (project, issue_id);
