ALTER TABLE sheet DISABLE TRIGGER update_sheet_updated_ts;

UPDATE sheet
SET payload = payload - 'type' - 'issueId' - 'issueName';

ALTER TABLE sheet ENABLE TRIGGER update_sheet_updated_ts;
