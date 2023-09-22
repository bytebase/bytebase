ALTER TABLE issue DISABLE TRIGGER update_issue_updated_ts;

UPDATE issue
SET type = 'bb.issue.database.general'
WHERE type IN ('bb.issue.database.create', 'bb.issue.database.schema.update', 'bb.issue.database.schema.update.ghost', 'bb.issue.database.data.update', 'bb.issue.database.restore.pitr');

ALTER TABLE issue ENABLE TRIGGER update_issue_updated_ts;
