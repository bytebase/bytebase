ALTER TABLE sheet DISABLE TRIGGER update_sheet_updated_ts;
ALTER TABLE task DISABLE TRIGGER update_task_updated_ts;

UPDATE task SET payload = payload - 'pushEvent';

UPDATE sheet
SET payload = payload || jsonb_build_object('vcsPayload', jsonb_build_object('pushEvent', (sheet.payload->'vcsPayload'->'pushEvent')::JSONB - 'fileCommit'))
WHERE payload->'vcsPayload' IS NOT NULL;

ALTER TABLE task ENABLE TRIGGER update_task_updated_ts;
ALTER TABLE sheet ENABLE TRIGGER update_sheet_updated_ts;
