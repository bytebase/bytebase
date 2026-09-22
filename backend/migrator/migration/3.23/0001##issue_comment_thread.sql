-- Add flat reply threads to issue comments.
ALTER TABLE issue_comment
    -- The root comment of this reply's thread; NULL on root comments and
    -- events. A reply references the root directly, never another reply.
    ADD COLUMN parent_id text REFERENCES issue_comment(resource_id),
    -- OPEN/RESOLVED on thread roots, the comments with a statement anchor;
    -- NULL on plain comments, replies, and events.
    ADD COLUMN thread_state text CHECK (thread_state IN ('OPEN', 'RESOLVED'));

-- FK referencing side and reply reads. Partial: most rows have no parent.
CREATE INDEX idx_issue_comment_parent_id ON issue_comment(parent_id)
    WHERE parent_id IS NOT NULL;

-- No backfill: legacy comments have no statement anchor, so they stay plain
-- comments outside threads.
CREATE INDEX idx_issue_comment_open_thread ON issue_comment(project, issue_id)
    WHERE thread_state = 'OPEN';
