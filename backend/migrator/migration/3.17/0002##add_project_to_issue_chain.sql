-- Phase A: Drop FKs
ALTER TABLE issue_comment DROP CONSTRAINT IF EXISTS issue_comment_issue_id_fkey;

-- Phase B: Change issue PK
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'issue_pkey'
          AND conrelid = 'issue'::regclass
          AND array_length(conkey, 1) = 2
    ) THEN
        ALTER TABLE issue DROP CONSTRAINT IF EXISTS issue_pkey;
        ALTER TABLE issue ADD PRIMARY KEY (project, id);
    END IF;
END $$;

-- Phase C: Modify issue_comment
-- Drop id column (unused — all lookups use resource_id)
DO $$ BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'issue_comment' AND column_name = 'id'
    ) THEN
        ALTER TABLE issue_comment DROP CONSTRAINT IF EXISTS issue_comment_pkey;
        ALTER TABLE issue_comment DROP COLUMN id;
    END IF;
END $$;

-- Add project column, backfill from issue
ALTER TABLE issue_comment ADD COLUMN IF NOT EXISTS project text;
UPDATE issue_comment SET project = issue.project FROM issue WHERE issue_comment.project IS NULL AND issue_comment.issue_id = issue.id;
DELETE FROM issue_comment WHERE project IS NULL;
ALTER TABLE issue_comment ALTER COLUMN project SET NOT NULL;

-- Use resource_id as PK
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'issue_comment_pkey'
          AND conrelid = 'issue_comment'::regclass
    ) THEN
        ALTER TABLE issue_comment ADD PRIMARY KEY (resource_id);
    END IF;
END $$;

-- Phase D: Re-add FKs and update indexes
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'issue_comment_issue_id_fkey' AND conrelid = 'issue_comment'::regclass) THEN
        ALTER TABLE issue_comment ADD CONSTRAINT issue_comment_issue_id_fkey
            FOREIGN KEY (project, issue_id) REFERENCES issue(project, id);
    END IF;
END $$;

DROP INDEX IF EXISTS idx_issue_comment_issue_id;
CREATE INDEX IF NOT EXISTS idx_issue_comment_issue_id ON issue_comment(project, issue_id);

DROP SEQUENCE IF EXISTS issue_comment_id_seq;
