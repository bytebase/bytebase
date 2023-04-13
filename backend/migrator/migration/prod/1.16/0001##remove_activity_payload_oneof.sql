UPDATE activity SET
    payload = payload->'issueCommentCreatePayload'
WHERE
    "type"='bb.issue.comment.create'
    AND
    payload ? 'issueCommentCreatePayload';