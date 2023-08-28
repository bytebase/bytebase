ALTER TABLE sheet DISABLE TRIGGER update_sheet_updated_ts;

UPDATE sheet
SET payload = (payload || jsonb_build_object('usedByIssues', jsonb_build_array(jsonb_build_object('issueId', CAST(payload->>'issueId' AS INTEGER), 'issueTitle', payload->>'issueName')))) - 'type' - 'issueId' - 'issueName'
WHERE payload ?& ARRAY['issueId','issueName'];

ALTER TABLE sheet ENABLE TRIGGER update_sheet_updated_ts;
