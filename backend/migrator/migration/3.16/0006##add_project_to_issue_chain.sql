-- Phase A: Drop FKs
ALTER TABLE issue_comment DROP CONSTRAINT issue_comment_issue_id_fkey;

-- Phase B: Change issue PK
ALTER TABLE issue DROP CONSTRAINT issue_pkey;
ALTER TABLE issue ADD PRIMARY KEY (project, id);

-- Phase C: Modify issue_comment
-- Drop id column (unused — all lookups use resource_id)
ALTER TABLE issue_comment DROP CONSTRAINT issue_comment_pkey;
ALTER TABLE issue_comment DROP COLUMN id;

-- Add project column, backfill from issue
ALTER TABLE issue_comment ADD COLUMN project text;
UPDATE issue_comment SET project = issue.project FROM issue WHERE issue_comment.issue_id = issue.id;
DELETE FROM issue_comment WHERE project IS NULL;
ALTER TABLE issue_comment ALTER COLUMN project SET NOT NULL;

-- Use resource_id as PK
ALTER TABLE issue_comment ADD PRIMARY KEY (resource_id);

-- Phase D: Re-add FKs and update indexes
ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_issue_id_fkey
    FOREIGN KEY (project, issue_id) REFERENCES issue(project, id);

DROP INDEX idx_issue_comment_issue_id;
CREATE INDEX idx_issue_comment_issue_id ON issue_comment(project, issue_id);

DROP SEQUENCE IF EXISTS issue_comment_id_seq;
