ALTER TABLE issue DISABLE TRIGGER update_issue_updated_ts;

UPDATE issue
SET payload = payload - 'grouping'
WHERE payload ? 'grouping';

ALTER TABLE issue ENABLE TRIGGER update_issue_updated_ts;
