-- Add flat reply threads to issue comments.
ALTER TABLE issue_comment
    -- The root comment of this reply's thread; NULL on root comments and
    -- events. A reply references the root directly, never another reply.
    ADD COLUMN parent_id text REFERENCES issue_comment(resource_id),
    -- OPEN/RESOLVED on root comments; NULL on replies and events.
    ADD COLUMN thread_state text CHECK (thread_state IN ('OPEN', 'RESOLVED'));

-- FK referencing side and reply reads. Partial: most rows have no parent.
CREATE INDEX idx_issue_comment_parent_id ON issue_comment(parent_id)
    WHERE parent_id IS NOT NULL;

-- Every valid legacy Comment becomes an OPEN thread root. Protojson omits an
-- empty comment, so both {} and {"comment": ""} are Comments. Event, hybrid,
-- legacy Event-key, and malformed payloads stay outside threads.
UPDATE issue_comment
SET thread_state = 'OPEN'
WHERE CASE WHEN jsonb_typeof(payload) = 'object'
           THEN payload - 'comment' = '{}'::jsonb
                AND (NOT (payload ? 'comment') OR jsonb_typeof(payload -> 'comment') = 'string')
           ELSE false END;

-- Built after the backfill: the UPDATE would otherwise maintain it per row.
CREATE INDEX idx_issue_comment_open_thread ON issue_comment(project, issue_id)
    WHERE thread_state = 'OPEN';

ANALYZE issue_comment;
